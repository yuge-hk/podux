import { useState, type FormEvent } from "react";
import {
  AlertDialog,
  Button,
  Callout,
  Dialog,
  Flex,
  Grid,
  Select,
  Switch,
  Text,
  TextArea,
  TextField,
} from "@radix-ui/themes";
import { useTranslation } from "react-i18next";
import { FormItem } from "../../components/FormItem";
import {
  configuration,
  defaults,
  errorKey,
  tunnelAPI,
  validConfig,
  type Capabilities,
  type Tunnel,
  type TunnelConfig,
} from "./api";

interface Props {
  tunnel?: Tunnel;
  caps?: Capabilities;
  onClose: () => void;
  onSaved: (tunnel: Tunnel, restart: boolean) => void;
}
export function TunnelForm({ tunnel, caps, onClose, onSaved }: Props) {
  const { t } = useTranslation();
  const [config, setConfig] = useState<TunnelConfig>(() =>
    tunnel ? configuration(tunnel) : { ...defaults }
  );
  const [revision, setRevision] = useState(tunnel?.revision ?? 0);
  const [token, setToken] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState(false);
  const set = <K extends keyof TunnelConfig>(key: K, value: TunnelConfig[K]) =>
    setConfig((c) => ({ ...c, [key]: value }));
  const select = <K extends "protocol" | "region" | "edgeIPVersion">(
    key: K,
    options: readonly TunnelConfig[K][]
  ) => (
    <FormItem label={t(`tunnels.fields.${key}`)} key={key}>
      <Select.Root
        value={config[key] || "default"}
        onValueChange={(value) => {
          const next = value === "default" ? "" : value;
          if (key === "protocol")
            setConfig((c) => ({
              ...c,
              protocol: next as TunnelConfig["protocol"],
              postQuantum: next === "quic" && c.postQuantum,
            }));
          else set(key, next as TunnelConfig[K]);
        }}
      >
        <Select.Trigger aria-label={t(`tunnels.fields.${key}`)} style={{ width: "100%" }} />
        <Select.Content>
          {options.map((value) => (
            <Select.Item key={value} value={value || "default"}>
              {value || t("tunnels.global")}
            </Select.Item>
          ))}
        </Select.Content>
      </Select.Root>
    </FormItem>
  );
  const number = (
    key: "maxRestartAttempts" | "connectionRetries" | "gracePeriodSeconds" | "metricsPort",
    min: number,
    max: number
  ) => (
    <FormItem key={key} label={t(`tunnels.fields.${key}`)} required>
      <TextField.Root
        aria-label={t(`tunnels.fields.${key}`)}
        type="number"
        min={min}
        max={max}
        step="1"
        required
        value={Number.isNaN(config[key]) ? "" : config[key]}
        onChange={(e) => set(key, e.target.value === "" ? NaN : Number(e.target.value))}
      />
      <Text as="p" size="1" color="gray">
        {t(`tunnels.help.${key}`)}
      </Text>
    </FormItem>
  );
  const toggle = (key: "autoStart" | "autoRestart" | "postQuantum", disabled = false) => (
    <Flex direction="column" gap="1" key={key}>
      <Text as="label" size="2">
        <Flex gap="2" align="center">
          <Switch
            checked={config[key]}
            disabled={disabled}
            aria-label={t(`tunnels.fields.${key}`)}
            onCheckedChange={(value) => set(key, value)}
          />
          {t(`tunnels.fields.${key}`)}
        </Flex>
      </Text>
      <Text size="1" color="gray">
        {t(`tunnels.help.${key}`)}
      </Text>
    </Flex>
  );
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (saving) return;
    if (!validConfig(config) || (!tunnel && !token.trim()) || /^(\*+|•+)$/.test(token.trim())) {
      setError("tunnels.errors.400");
      return;
    }
    setSaving(true);
    setError("");
    setConflict(false);
    try {
      if (tunnel) {
        const result = await tunnelAPI.update({ ...tunnel, revision }, config, token);
        setToken("");
        onSaved(result.tunnel, result.restartRequired);
      } else {
        const result = await tunnelAPI.create(config, token.trim());
        setToken("");
        onSaved(result, false);
      }
    } catch (e) {
      const key = errorKey(e, true);
      setError(key);
      setConflict(key === "tunnels.errors.revision");
    } finally {
      setSaving(false);
    }
  };
  const reload = async () => {
    if (!tunnel) return;
    setSaving(true);
    try {
      const latest = await tunnelAPI.get(tunnel.id);
      setConfig(configuration(latest));
      setRevision(latest.revision);
      setToken("");
      setConflict(false);
      setError("");
    } catch (e) {
      setError(errorKey(e));
    } finally {
      setSaving(false);
    }
  };
  const supported = (flag: string) => !caps?.available || caps.flags.includes(flag);
  return (
    <Dialog.Root
      open
      onOpenChange={(open) => {
        if (!open && !saving) {
          setToken("");
          onClose();
        }
      }}
    >
      <Dialog.Content className="tunnel-dialog" maxWidth="640px">
        <Dialog.Title>{t(tunnel ? "tunnels.edit" : "tunnels.add")}</Dialog.Title>
        <Dialog.Description size="2" mb="4">
          {t("tunnels.formDescription")}
        </Dialog.Description>
        <form onSubmit={submit}>
          <Flex direction="column" gap="4">
            {error && (
              <Callout.Root color="red" role="alert">
                <Callout.Text>{t(error)}</Callout.Text>
              </Callout.Root>
            )}
            {conflict && (
              <AlertDialog.Root>
                <AlertDialog.Trigger>
                  <Button type="button" variant="soft">
                    {t("tunnels.reload")}
                  </Button>
                </AlertDialog.Trigger>
                <AlertDialog.Content>
                  <AlertDialog.Title>{t("tunnels.reload")}</AlertDialog.Title>
                  <AlertDialog.Description>{t("tunnels.discard")}</AlertDialog.Description>
                  <Flex gap="3" justify="end" mt="4">
                    <AlertDialog.Cancel>
                      <Button variant="soft">{t("tunnels.cancel")}</Button>
                    </AlertDialog.Cancel>
                    <AlertDialog.Action>
                      <Button onClick={() => void reload()}>{t("tunnels.reload")}</Button>
                    </AlertDialog.Action>
                  </Flex>
                </AlertDialog.Content>
              </AlertDialog.Root>
            )}
            <FormItem label={t("tunnels.fields.name")} required>
              <TextField.Root
                aria-label={t("tunnels.fields.name")}
                value={config.name}
                required
                onChange={(e) => set("name", e.target.value)}
              />
            </FormItem>
            <FormItem label={t("tunnels.fields.description")}>
              <TextArea
                aria-label={t("tunnels.fields.description")}
                value={config.description}
                onChange={(e) => set("description", e.target.value)}
              />
            </FormItem>
            <FormItem label={t("tunnels.fields.token")} required={!tunnel}>
              <TextField.Root
                type="password"
                autoComplete="new-password"
                spellCheck={false}
                aria-label={t("tunnels.fields.token")}
                value={token}
                required={!tunnel}
                onChange={(e) => setToken(e.target.value)}
              />
              <Text as="p" size="1" color="gray">
                {t(tunnel?.hasToken ? "tunnels.tokenSaved" : "tunnels.tokenHelp")}
              </Text>
            </FormItem>
            {toggle("autoStart")}
            {toggle("autoRestart")}
            {number("maxRestartAttempts", 0, 100)}
            <details>
              <summary>{t("tunnels.advanced")}</summary>
              <Grid columns={{ initial: "1", sm: "2" }} gap="4" mt="4">
                {select("protocol", ["auto", "quic", "http2"])}
                {select("region", ["", "us"])}
                {select("edgeIPVersion", ["auto", "4", "6"])}
                <FormItem label={t("tunnels.fields.edgeBindAddress")}>
                  <TextField.Root
                    aria-label={t("tunnels.fields.edgeBindAddress")}
                    value={config.edgeBindAddress}
                    onChange={(e) => set("edgeBindAddress", e.target.value.trim())}
                  />
                  <Text size="1" color="gray">
                    {t("tunnels.help.edgeBindAddress")}
                  </Text>
                </FormItem>
                {toggle("postQuantum", config.protocol !== "quic" || !supported("post-quantum"))}
                {number("connectionRetries", 0, 20)}
                {number("gracePeriodSeconds", 1, 120)}
                {number("metricsPort", 0, 65535)}
                <FormItem label={t("tunnels.fields.haConnections")}>
                  <Select.Root
                    value={
                      config.extraArgs.find((a) => a.name === "ha-connections")?.value ?? "default"
                    }
                    onValueChange={(value) =>
                      set("extraArgs", [
                        ...config.extraArgs.filter((a) => a.name !== "ha-connections"),
                        ...(value === "default"
                          ? []
                          : [{ name: "ha-connections" as const, value }]),
                      ])
                    }
                    disabled={!supported("ha-connections")}
                  >
                    <Select.Trigger aria-label={t("tunnels.fields.haConnections")} />
                    <Select.Content>
                      {["default", "1", "2", "3", "4"].map((v) => (
                        <Select.Item key={v} value={v}>
                          {v === "default" ? t("tunnels.default") : v}
                        </Select.Item>
                      ))}
                    </Select.Content>
                  </Select.Root>
                </FormItem>
                <Text as="label" size="2">
                  <Flex gap="2" align="center">
                    <Switch
                      aria-label={t("tunnels.fields.loglevel")}
                      checked={config.extraArgs.some((a) => a.name === "loglevel")}
                      disabled={!supported("loglevel")}
                      onCheckedChange={(checked) =>
                        set("extraArgs", [
                          ...config.extraArgs.filter((a) => a.name !== "loglevel"),
                          ...(checked ? [{ name: "loglevel" as const, value: "info" }] : []),
                        ])
                      }
                    />
                    {t("tunnels.fields.loglevel")}
                  </Flex>
                </Text>
              </Grid>
              <Text as="p" size="1" color="gray" mt="3">
                {t("tunnels.extraHelp")}
              </Text>
            </details>
            <Flex justify="end" gap="3" wrap="wrap">
              <Button
                type="button"
                variant="soft"
                disabled={saving}
                onClick={() => {
                  setToken("");
                  onClose();
                }}
              >
                {t("tunnels.cancel")}
              </Button>
              <Button type="submit" loading={saving}>
                {t("tunnels.save")}
              </Button>
            </Flex>
          </Flex>
        </form>
      </Dialog.Content>
    </Dialog.Root>
  );
}

import { useState } from "react";
import { AlertDialog, Button, Callout, Flex } from "@radix-ui/themes";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { errorKey, tunnelAPI, type Action, type Tunnel } from "./api";
import { useTunnels } from "./hooks";
import { TunnelsPageView } from "./TunnelsPage.view";
import { TunnelForm } from "./TunnelForm";
import { TunnelDetail } from "./TunnelDetail";
import "./Tunnels.css";
export function TunnelsPage({ embedded = false }: { embedded?: boolean }) {
  const { t } = useTranslation();
  const state = useTunnels();
  const [editing, setEditing] = useState<Tunnel | "new">();
  const [detail, setDetail] = useState<Tunnel>();
  const [deleting, setDeleting] = useState<Tunnel>();
  const [deleteBusy, setDeleteBusy] = useState(false);
  const [deleteError, setDeleteError] = useState("");
  const [restart, setRestart] = useState<Tunnel>();
  const [restartError, setRestartError] = useState("");
  const action = async (id: string, command: Action) => {
    try {
      await state.control(id, command);
      toast.success(t("tunnels.accepted"));
    } catch (e) {
      toast.error(t(errorKey(e)));
    }
  };
  const remove = async () => {
    if (!deleting || deleteBusy) return;
    setDeleteBusy(true);
    setDeleteError("");
    try {
      await tunnelAPI.remove(deleting);
      setDeleting(undefined);
      await state.refresh();
      toast.success(t("tunnels.deleted"));
    } catch (e) {
      setDeleteError(errorKey(e, true));
    } finally {
      setDeleteBusy(false);
    }
  };
  return (
    <>
      <TunnelsPageView
        {...state}
        embedded={embedded}
        onAdd={() => setEditing("new")}
        onEdit={setEditing}
        onDetail={setDetail}
        onDelete={(item) => {
          setDeleteError("");
          setDeleting(item);
        }}
        onAction={(id, a) => void action(id, a)}
        onRefresh={() => void state.refresh()}
      />
      {editing && (
        <TunnelForm
          key={editing === "new" ? "new" : editing.id}
          tunnel={editing === "new" ? undefined : editing}
          caps={state.caps}
          onClose={() => setEditing(undefined)}
          onSaved={(item, required) => {
            setEditing(undefined);
            void state.refresh();
            if (required) {
              setRestart(item);
              setRestartError("");
            } else toast.success(t("tunnels.saved"));
          }}
        />
      )}
      {detail && (
        <TunnelDetail
          tunnel={state.items.find((i) => i.id === detail.id) ?? detail}
          status={state.statuses[detail.id]}
          onClose={() => setDetail(undefined)}
        />
      )}
      <AlertDialog.Root
        open={!!deleting}
        onOpenChange={(open) => {
          if (!open && !deleteBusy) setDeleting(undefined);
        }}
      >
        <AlertDialog.Content className="tunnel-dialog">
          <AlertDialog.Title>{t("tunnels.deleteTitle")}</AlertDialog.Title>
          <AlertDialog.Description>{t("tunnels.deleteHelp")}</AlertDialog.Description>
          {deleteError && (
            <Callout.Root color="red" role="alert" mt="3">
              <Callout.Text>{t(deleteError)}</Callout.Text>
            </Callout.Root>
          )}
          <Flex gap="3" justify="end" mt="4" wrap="wrap">
            <AlertDialog.Cancel>
              <Button variant="soft" disabled={deleteBusy}>
                {t("tunnels.cancel")}
              </Button>
            </AlertDialog.Cancel>
            <Button color="red" loading={deleteBusy} onClick={() => void remove()}>
              {t("tunnels.delete")}
            </Button>
          </Flex>
        </AlertDialog.Content>
      </AlertDialog.Root>
      <AlertDialog.Root
        open={!!restart}
        onOpenChange={(open) => {
          if (!open) setRestart(undefined);
        }}
      >
        <AlertDialog.Content className="tunnel-dialog">
          <AlertDialog.Title>{t("tunnels.saved")}</AlertDialog.Title>
          <AlertDialog.Description>{t("tunnels.restartHelp")}</AlertDialog.Description>
          {restartError && (
            <Callout.Root color="red" role="alert">
              <Callout.Text>{t(restartError)}</Callout.Text>
            </Callout.Root>
          )}
          <Flex gap="3" justify="end" mt="4" wrap="wrap">
            <AlertDialog.Cancel>
              <Button variant="soft">{t("tunnels.later")}</Button>
            </AlertDialog.Cancel>
            <Button
              disabled={!state.caps?.available}
              loading={restart ? state.pending[restart.id] : false}
              onClick={async () => {
                if (!restart) return;
                try {
                  await state.control(restart.id, "restart");
                  setRestart(undefined);
                } catch (e) {
                  setRestartError(errorKey(e));
                }
              }}
            >
              {t("tunnels.now")}
            </Button>
          </Flex>
        </AlertDialog.Content>
      </AlertDialog.Root>
    </>
  );
}

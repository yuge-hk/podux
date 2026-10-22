# Draft: UDPProxy.InWorkConn channel-field races during work-connection replacement in v0.71.1

Observed in Linux amd64 trial 1 and macOS arm64 trials 1 and 2. Overlapping InWorkConn calls write readCh/sendCh at client/proxy/udp.go:111–112 while another call reads those instance fields for udp.Forwarder at :168. The writes occur under the proxy mutex; these reads do not share that lock.

Expected: replacement/reconnection of a UDP work connection does not cause race detector reports. Both channel-field access pairs are reported. The same field pattern appeared previously in a v0.68.0 feature-branch run; this v0.71.1 reproduction contains no Cloudflare module.

Actual access stacks (Linux amd64 trial 1):

```text
WARNING: DATA RACE
Write at 0x00c000433198 by goroutine 521:
  github.com/fatedier/frp/client/proxy.(*UDPProxy).InWorkConn()
      /home/runner/go/pkg/mod/github.com/fatedier/frp@v0.71.1/client/proxy/udp.go:111 +0x533
  github.com/fatedier/frp/client/proxy.(*Wrapper).InWorkConn.gowrap1()
      /home/runner/go/pkg/mod/github.com/fatedier/frp@v0.71.1/client/proxy/proxy_wrapper.go:274 +0x66

Previous read at 0x00c000433198 by goroutine 515:
  github.com/fatedier/frp/client/proxy.(*UDPProxy).InWorkConn()
      /home/runner/go/pkg/mod/github.com/fatedier/frp@v0.71.1/client/proxy/udp.go:168 +0xacf
  github.com/fatedier/frp/client/proxy.(*Wrapper).InWorkConn.gowrap1()
      /home/runner/go/pkg/mod/github.com/fatedier/frp@v0.71.1/client/proxy/proxy_wrapper.go:274 +0x66

WARNING: DATA RACE
Write at 0x00c0004331a0 by goroutine 521:
  github.com/fatedier/frp/client/proxy.(*UDPProxy).InWorkConn()
      /home/runner/go/pkg/mod/github.com/fatedier/frp@v0.71.1/client/proxy/udp.go:112 +0x5a9
  github.com/fatedier/frp/client/proxy.(*Wrapper).InWorkConn.gowrap1()
      /home/runner/go/pkg/mod/github.com/fatedier/frp@v0.71.1/client/proxy/proxy_wrapper.go:274 +0x66

Previous read at 0x00c0004331a0 by goroutine 515:
  github.com/fatedier/frp/client/proxy.(*UDPProxy).InWorkConn()
      /home/runner/go/pkg/mod/github.com/fatedier/frp@v0.71.1/client/proxy/udp.go:168 +0xaf0
  github.com/fatedier/frp/client/proxy.(*Wrapper).InWorkConn.gowrap1()
      /home/runner/go/pkg/mod/github.com/fatedier/frp@v0.71.1/client/proxy/proxy_wrapper.go:274 +0x66
```

[Complete original output](https://github.com/yuge-hk/podux/blob/test/frp-0711-race-20261023/development/phase3-evidence/frp-0711-validation/native/linux-amd64/frpc-race-1.txt).


Environment: official Podux main `1136797a36a2f1186055278351f55e28b907d6d5`, with only FRP upgraded to v0.71.1 and Go pinned to 1.26.10 in a disposable branch; no FRPC implementation patch. The FRPC integration test is byte-identical to the previous v0.68.0 control. Only loopback TCP/UDP origins/FRPS and synthetic local credentials are used.

Tested commit: `8ae7bd98818946262a3f830d3279d03eb18e9047`.
[Native CI and artifacts](https://github.com/yuge-hk/podux/actions/runs/39851903747).
[Exact test source](https://github.com/yuge-hk/podux/blob/8ae7bd98818946262a3f830d3279d03eb18e9047/internal/integration/frpc_regression_test.go).

Reproduction on Linux with Go 1.26.10, Python 3 and a CGO compiler:

```sh
git clone --branch test/frp-0711-race-20261023 https://github.com/yuge-hk/podux.git podux-frp-race
cd podux-frp-race
git checkout 8ae7bd98818946262a3f830d3279d03eb18e9047
python3 build/ci/build_frps.py --output /tmp/podux-frps-0711 > /tmp/podux-frps-provenance.json
export PODUX_TEST_FRPS=/tmp/podux-frps-0711
CGO_ENABLED=1 go test -race -json -count=1 -timeout 3m ./internal/integration -run '^TestFRPC'
```

The scenario verifies TCP/UDP forwarding, proxy disable/restore via existing ReloadFrpc, FRPS stop/restart, reconnect, client shutdown and port release. Functional runs pass. Race detection is nondeterministic: CI runs three predetermined identical invocations per platform and preserves every failure/pass; any failure fails the job. Complete goroutine creation stacks are retained in the linked evidence. No production service is required.

The previous controller stop/keepControllerWorking race is not the subject of this report: v0.71.1 contains the lock-based repair and it did not recur in the 12 new native trials. Existing lifecycle Issue #5557 / PRs #5547 and #5560 were reviewed. Please check for duplicate current reports before submitting this draft.

This is a local, unpublished draft. No Issue, comment or PR has been sent.

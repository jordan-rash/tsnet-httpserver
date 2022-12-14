# Tailscale HTTPServer wasmCloud Provider

Provider that utilizes the [tsnet](https://github.com/tailscale/tailscale/tree/main/tsnet) Go module to expose a linked actor to a users Tailnet.

Can be used with any actor that satisfies the `wasmcloud:httpserver` contract

### Link Definition Settings
| Setting    | Type   | Default | Required |
| ---------- | ------ | ------- | -------- |
| port       | int    | None    | true     |
| hostname   | string | None    | true     |
| ts_authkey | string | None    | true     |

> Recommended that you use a pre-authorized tailscale key and tag it with "tag:wasmcloud" or something similiar

### Artifacts
Usable OCI artifacts can be found in Github Packages (linked in the right hand side bar)

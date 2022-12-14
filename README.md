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

Once you actor has been successfully linked, you should see the `hostname` appear in your Tailscale admin console.

<img width="1175" alt="Screenshot 2022-12-14 at 2 11 23 PM" src="https://user-images.githubusercontent.com/15827604/207715197-06ea8f16-d1b5-45b7-92a2-20196c767ee8.png">

At this point, if you are on a machine that is connected to your tailnet, you can visit the IP or URL to see your actor!

![image](https://user-images.githubusercontent.com/15827604/207715468-3f6d0bf7-edb7-4434-8a14-5d77d391a0a3.png)

### Artifacts
Usable OCI artifacts can be found in Github Packages (linked in the right hand side bar)

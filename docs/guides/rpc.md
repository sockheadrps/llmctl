# Distributed GPU with RPC

RPC mode lets you pool GPU resources across two machines. One machine contributes
its GPU over the network; the other runs the model and offloads some layers to it,
effectively running a model larger than either machine could hold alone.

---

## On This Page

- [Distributed GPU with RPC](#distributed-gpu-with-rpc)
  - [On This Page](#on-this-page)
  - [How It Works](#how-it-works)
  - [Prerequisites](#prerequisites)
  - [Step 1: Set Up the Server Machine](#step-1-set-up-the-server-machine)
  - [Step 2: Set Up the Client Machine](#step-2-set-up-the-client-machine)
  - [Step 3: Verify the Connection](#step-3-verify-the-connection)
  - [Step 4: Start a Model with RPC](#step-4-start-a-model-with-rpc)
  - [Troubleshooting](#troubleshooting)
  - [Direct Ethernet MTU Tuning](#direct-ethernet-mtu-tuning)

## How It Works

llmctl wraps [llama.cpp's RPC backend](https://github.com/ggerganov/llama.cpp/tree/master/examples/rpc). The setup is two roles:

**RPC server machine** - starts a `ggml-rpc-server` process that exposes its GPU over the network. It doesn't run models itself; it just provides VRAM capacity.

**RPC client machine** - runs `llama-server` with some layers offloaded to the RPC server. From the model's perspective, it has access to both GPUs. This machine also runs llmctl normally.

Only the client machine needs llmctl. The server machine only needs the `ggml-rpc-server` binary (and optionally llmctl to manage it).

RPC mode is separate from the local status server. You can enable the status server
and dashboard without RPC, or run RPC without turning on the browser dashboard.

---

## Prerequisites

Both machines need:

- The same version of `llama-server` / `ggml-rpc-server` - mismatched versions are the most common source of connection failures.
- Network connectivity between them - they need to reach each other on the RPC port (default `50052`).

---

## Step 1: Set Up the Server Machine

On the machine that will contribute its GPU:

1. Open llmctl and go to **Settings**
2. Enable **RPC mode** and set it to **Server**
3. Optionally configure the bind host and port
4. Go to the **RPC Server** tab

![RPC Server start modal in server mode](/assets/screenshots/rpcserverstartmodal.png)

Press `Enter` on the **Start** row to launch `ggml-rpc-server`. The status changes to **ONLINE**.

![RPC Server tab showing ONLINE status and LAN addresses](/assets/screenshots/rpcserverup.png)

Under **LAN addresses** you'll see the local IPs where the status server is reachable. Press `Enter` on one to copy it - you'll paste this into the client machine's Settings in the next step.

---

## Step 2: Set Up the Client Machine

On the machine that will run the model:

1. Open llmctl and go to **Settings**
2. Enable **RPC mode** and set it to **Client**
3. Under **Remote status server**, paste the address you copied from the server machine (format: `192.168.x.x:11435`)

llmctl polls that address and auto-discovers the RPC endpoint. You don't need to enter the RPC port manually.

---

## Step 3: Verify the Connection

Go to the **RPC Server** tab on the client machine. The status should show **CONNECTED** with the discovered RPC endpoint address.

![Client machine connected to a remote RPC server](/assets/screenshots/rpcclientconnection.png)

Switch to the **Overview** tab. Under **System Telemetry**, you'll see:
- **GPU 0** - your local GPU name and VRAM
- **GPU 1** - the remote machine's GPU name and VRAM

![Overview tab with local and remote telemetry populated](/assets/screenshots/concepts-overview-populated.png)

If the model is split across GPUs, the Overview and dashboard also show per-GPU model-load slices once the startup log has been parsed.

If GPU 1 doesn't appear, the client hasn't successfully connected yet - see [Troubleshooting](#troubleshooting) below.

---

## Step 4: Start a Model with RPC

Create or select a profile on the client machine. With RPC connected, `llama-server` automatically uses the remote GPU for the layers you've allocated.

The number of GPU layers in your profile is split across both GPUs. llama.cpp decides the distribution based on available VRAM. Load time will be slightly longer than local-only because weights are transferred over the network, but generation speed is typically close to having both GPUs locally.

Once running, the **Overview** tab on the server machine (if it's also running llmctl) will show the client's running instances under **Remote** in Active Services. The browser dashboard uses the same data.

---

## Troubleshooting

**RPC client shows "polling..." or "not configured"**

- Double-check the status server address in Settings - it should be `host:port`, not just a host
- Make sure the server machine's firewall allows the status server port (default `11435`)
- Confirm `ggml-rpc-server` is actually running on the server (RPC Server tab -> status should be ONLINE)

**Model fails to start or crashes immediately**

- The most common cause: mismatched llama.cpp versions. The `llama-server` on the client and `ggml-rpc-server` on the server must be from the same release.
- Check the log viewer (`e`) for connection refused errors pointing to the RPC port

---

## Direct Ethernet MTU Tuning

**GPU 1 shows in telemetry but model is slow**

- High network latency between machines will slow down per-token generation. RPC works best on a local gigabit network.
- Check that you're not saturating the link - running other heavy network traffic simultaneously will hurt performance.
- If the machines are linked directly by Ethernet, raise the MTU on both NICs. Jumbo frames cut packet overhead during large tensor transfers, and they matter more on a direct cable than on a busy LAN. Keep the MTU identical at both ends, and make sure both adapters actually support the larger frame size.

  On Windows, that usually means enabling the NIC's Jumbo Packet setting first, then setting the adapter MTU to `9000` or the closest supported value. On Linux, use `ip link` or NetworkManager to set the same MTU on both the RPC client and server interfaces.


# Local Models

How to import GGUF models, start and stop them, and work with multiple instances at once.

---

## The Models Tab

The **Models** tab is where you manage everything related to your local GGUF files. Open it by pressing `d` from the Overview tab.

The left pane shows a tree of all registered models. Each model can be expanded to reveal its profiles.

![Models tab showing the tree with models and profiles](../assets/screenshots/local-models-tree.png)

---

## Pointing llmctl at Your GGUF Files

Before any models appear, llmctl needs to know where to look.

1. Go to the **Settings** tab (`d` twice from Overview)
2. Under **Model directories**, press `Enter` on **+ Add directory** and enter a folder path
3. llmctl scans that folder and makes every `.gguf` file available to import

You can add as many directories as you want. llmctl scans all of them.

---

## Importing a Model

With your model directory set:

1. Go to the **Models** tab
2. Press `Enter` on **+ Import Model** at the bottom of the list (or the import prompt that appears when no models are registered yet)
3. A picker shows every `.gguf` file found in your configured directories — select one and confirm

The model appears in the tree. It won't have any profiles yet.

---

## Creating a Profile

A model without a profile can't run. Press `Enter` or `→` to expand the model, then navigate to **+ New Profile** and press `Enter`.

The new profile form opens. Fill in at minimum:

| Field | What it controls |
|---|---|
| **Port** | The port `llama-server` will listen on for API requests |

Everything else has defaults. Press `Enter` to save.

The profile appears under the model. You can create as many profiles as you want — different ports, different GPU allocations, different context sizes.

See the [Profiles guide](./profiles) for a full explanation of what each setting does.

---

## Starting a Model

Navigate to a Model, drop into its profiles and press `Enter`. A confirmation screen shows what will launch — confirm to start `llama-server` with that profile.

The health dot next to the profile changes:

| Dot | Meaning |
|---|---|
| Yellow | Loading — `llama-server` is reading the weights into GPU/RAM |
| Green | Up — the model is accepting requests |
| Red | Down — the process exited or failed to start |

Load time depends on model size and hardware. A 4GB Q4 model on a modern GPU typically loads in 5–15 seconds. CPU-only loads take longer.

---

## Confirming It's Running

Once the dot turns green, check the **Overview** tab — your model appears under **Local** in the Active Services box with its alias, size, GPU/CPU badge, uptime, port, and live tok/s statistics. 

You can also switch to the **Running** tab for a more detailed live view showing the rate meter and VRAM usage. 

![Running tab showing a model with green health dot, VRAM bar, and rate meter](../assets/screenshots/local-models-running.png)

---

## Stopping a Model

Navigate to the running profile row (it shows the health dot) and press `Enter`. A stop confirmation appears. Confirm to terminate the `llama-server` process.

The profile remains — only the running instance is stopped. You can start it again any time.

You can also stop from the **Running** tab: press `Enter` on any instance row to open the action menu and choose **Stop**.

---

## Running Multiple Models at Once

You can run as many models simultaneously as your hardware allows, as long as each is on a different port. llmctl starts and tracks each as a separate instance.

Things to watch:
- **VRAM** — each GPU-loaded model consumes VRAM. The Running tab shows a VRAM bar across all active processes.
- **RAM** — CPU-only models consume system RAM. System Telemetry in the Overview tab shows aggregate RAM used by CPU-only processes.
- **Ports** — if two profiles share a port, the second will fail to start. Pick distinct ports for each profile.
- **RPC** Only one model can load on RPC at a time. The llama cpp RPC server that facilitates the RPC connection is what limits this functionality.

---

## When a Model Fails to Start

If the health dot turns red immediately or never leaves yellow after a long wait:

1. Press `e` from anywhere on the main screen — this opens the log viewer for the most recently failed instance
2. The log shows `llama-server` stdout/stderr output — look for error messages about VRAM, missing files, or unsupported model formats

Common causes:
- Not enough VRAM — lower **GPU Layers** in the profile
- Wrong binary path — check **llama-server binary** in Settings
- Model file is corrupt or unsupported format — try a different GGUF

You can also view logs for a specific instance from the **Running** tab: press `Enter` on an instance row and choose **View logs**.

---

## Deleting a Profile or Model

Navigate to the row and press `Del`. A confirmation prompt appears — press `Del` again to confirm.

Deleting a model removes all its profiles. Deleting a profile only removes that configuration; the model and its other profiles remain.

You can't delete a model or profile while it has a running instance. Stop the instance first.

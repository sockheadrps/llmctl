package tui

import "github.com/charmbracelet/bubbles/textinput"

// settingsCategoryDef is one entry in the Settings tab's category list. id
// is the stable key selectRow/enterSettingsCategory use to route to it;
// label is what's displayed.
type settingsCategoryDef struct {
	id    string
	label string
}

// settingsCategories is the single source of truth for the Settings tab's
// left-pane category list (via buildSettingsRows). Add a new settings page
// by appending here and giving renderSettingsDetail a case for its id.
var settingsCategories = []settingsCategoryDef{
	{id: "model_dirs", label: "Model Directories"},
	{id: "llama_bin", label: "llama-server Binary"},
	{id: "rpc", label: "RPC"},
	{id: "status_server", label: "Status Server"},
}

// settingsState backs the Settings tab's content — currently just Model
// Directories, but a struct (rather than dirsContentState directly) leaves
// room to add another category's state alongside it later.
type rpcContentState struct {
	// cursor positions:
	// 0 = toggle enabled
	// 1 = select Client mode
	// 2 = select Server mode
	// when client: 3 = remote status addr, 4 = manual endpoint
	// when server: 3 = status host, 4 = status port, 5 = firewall/network,
	// 6 = RPC binary (Windows), 7 = RPC port (Windows)
	cursor            int
	remoteAddrEditing bool
	remoteAddrInput   textinput.Model
	editing           bool
	input             textinput.Model
	rpcBinEditing     bool
	rpcBinInput       textinput.Model
	portEditing       bool
	portInput         textinput.Model
	err               string
}

type statusServerContentState struct {
	// cursor positions:
	// 0 = toggle status server
	// 1 = edit host
	// 2 = edit port
	// 3 = toggle history persistence
	// 4 = toggle dashboard availability
	// 5 = copy firewall rule (Windows only, when enabled)
	cursor      int
	hostEditing bool
	hostInput   textinput.Model
	portEditing bool
	portInput   textinput.Model
	copied      bool // true briefly after copying the firewall rule
	err         string
}

type settingsState struct {
	activeCategory string
	dirs           dirsContentState
	bin            binContentState
	rpc            rpcContentState
	statusSrv      statusServerContentState
}

// dirsContentState is the Model Directories category's content: the
// configured directory list (row 0 is always "+ Add Directory", rows
// 1..N are list[0..N-1]) plus an inline add/edit form.
type dirsContentState struct {
	list       []string
	cursor     int // 0 = "+ Add Directory"; i = list[i-1] for i>0
	pendingDel string

	editing    bool
	editingIdx int // index into list being edited; -1 while adding new
	input      textinput.Model
	err        string
}

type binContentState struct {
	editing bool
	input   textinput.Model
	err     string
}

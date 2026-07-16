package statusserver

import _ "embed"

// dashboardHTML contains the embedded browser dashboard.
//
//go:embed dashboard.html
var dashboardHTML string

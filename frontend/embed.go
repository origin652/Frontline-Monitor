package frontend

import "embed"

// Assets embeds the standalone frontend entry and static assets.
//
//go:embed index.html app.js styles.css
var Assets embed.FS

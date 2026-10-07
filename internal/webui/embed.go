package webui

import _ "embed"

//go:embed index.html
var Index []byte

//go:embed app.js
var AppJS []byte

//go:embed styles.css
var Styles []byte

package kubernetes

// fieldManagerName is the field manager used for server-side apply.
const fieldManagerName = "opm-cli"

// FieldManager is the field manager every server-side apply of the CLI
// writes as.
const FieldManager = fieldManagerName

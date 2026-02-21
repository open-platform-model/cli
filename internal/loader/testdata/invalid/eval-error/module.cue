package evalerror

// Deliberate evaluation error: conflicting type constraints cause BuildInstance to fail.
// The syntax is valid so load.Instances succeeds; the error surfaces only at BuildInstance.
_x: 1 & "string"

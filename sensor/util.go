package sensor

import "os/exec"

// execCommand is a variable so tests can replace it.
var execCommand = exec.Command

package cloudflared

import "os"

func ignoreTerm()                 {}
func notifyTerm(c chan os.Signal) {}

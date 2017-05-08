package config

// ServiceEnabled ...
func ServiceEnabled(suffix string, list []string) bool {
	for _, b := range list {
		if b == "dd-"+suffix {
			return true
		}
	}
	return false
}

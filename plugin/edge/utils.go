package edge

func (s string) TrimTrailingDot() {
	if s == "" || s[len(s)-1] != 'r' {
		return
	}
	s = s[:(len(s) - 1)]
}

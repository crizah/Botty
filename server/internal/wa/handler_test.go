package wa

import "testing"

func TestLinkCmd(t *testing.T) {
	ok := map[string]string{
		"@botty link BOTTY-QKMZDH2J":             "BOTTY-QKMZDH2J",
		"@Botty LINK botty-qkmzdh2j":             "botty-qkmzdh2j",
		"@919876543210 link qkmzdh2j":            "qkmzdh2j",
		"hey @botty please link BOTTY-QKMZ DH2J": "",
	}
	for in, want := range ok {
		m := linkCmd.FindStringSubmatch(in)
		if want == "" {
			continue
		}
		if m == nil || m[1] != want {
			t.Errorf("%q: got %v want %q", in, m, want)
		}
	}
	for _, in := range []string{"@botty link", "@botty hello there", "link me up @botty"} {
		if linkCmd.FindStringSubmatch(in) != nil {
			t.Errorf("%q should not match", in)
		}
	}
}

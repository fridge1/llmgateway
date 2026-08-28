package invoice

import "testing"

func TestUSCCPattern(t *testing.T) {
	valid := []string{
		"91110108MA01ABCD2X", // typical USCC shape
		"913100005960029871",
	}
	for _, v := range valid {
		if !usccPattern.MatchString(v) {
			t.Errorf("expected valid USCC: %s", v)
		}
	}
	invalid := []string{
		"",
		"123",
		"91110108MA01ABCD2",   // 17 chars
		"91110108MA01ABCD2XX", // 19 chars
		"9111010ima01abcd2x",  // lowercase
		"91110108MA01ABIO2X",  // contains I and O (excluded charset)
	}
	for _, v := range invalid {
		if usccPattern.MatchString(v) {
			t.Errorf("expected invalid USCC: %s", v)
		}
	}
}

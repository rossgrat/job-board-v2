package icims

import "testing"

func TestFormatLocations(t *testing.T) {
	cases := []struct{ in, want string }{
		{`[{"@type":"Place","address":{"@type":"PostalAddress","postalCode":"60714","addressRegion":"IL","streetAddress":"5800 West Touhy Avenue","addressCountry":"US","addressLocality":"Niles","postOfficeBoxNumber":"UNAVAILABLE"}}]`, "Niles, IL, US"},
		{`{"@type":"Place","address":{"addressLocality":"Niles","addressRegion":"IL","addressCountry":"US"}}`, "Niles, IL, US"},
		{`[{"address":{"addressLocality":"Niles","addressRegion":"IL","addressCountry":"US"}},{"address":{"addressLocality":"Boston","addressRegion":"MA","addressCountry":"US"}}]`, "Niles, IL, US; Boston, MA, US"},
		{`[{"address":{"addressCountry":"US"}}]`, "US"},
		{`"garbage"`, `"garbage"`},
	}
	for i, c := range cases {
		got := formatLocations([]byte(c.in))
		if got != c.want {
			t.Errorf("case %d: got %q want %q", i, got, c.want)
		}
	}
}

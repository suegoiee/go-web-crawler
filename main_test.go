package main

import (
	"testing"
)

// func TestLinkFilter(t *testing.T) {
// 	cases := []struct {
// 		name string
// 		href string
// 		want bool
// 	}{
// 		{name: "crorrect 1", href: "https://tw.stock.yahoo.com/news/q2%E5%95%86%E8%BE%A6%EF%BC%81a%E8%BE%A6%E7%A9%BA%E7%BD%AE%E7%8E%8744%EF%BC%85%EF%BC%8C%E7%A7%9F%E9%87%91qoq%E5%BE%AE%E5%8D%8706%EF%BC%85-064152206.html", want: true},
// 		{name: "wrong 1", href: "https://www.momoshop.com.tw/category/DgrpCategory.jsp?d_code=4300100677&p_orderType=4&showType=chessboardType&osm=googleKw&utm_source=google&utm_medium=cpc5-eCPC", want: false},
// 		{name: "wrong 2", href: "/living/378672", want: false},
// 		{name: "crorrect 2", href: "/news/living/378672", want: true},
// 	}
// 	for _, tt := range cases {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := linkFilter(tt.href); got != tt.want {
// 				t.Errorf("%v is wrong", tt.name)
// 			}
// 		})
// 	}
// }

func TestLinkFilter(t *testing.T) {
	tests := []struct {
		name     string
		href     string
		expected bool
	}{
		// Test cases where both linkExists and domainExists are true
		{name: "Valid News Link with HTTPS", href: "https://example.com/news/article", expected: true},
		{name: "Valid News Link without HTTPS", href: "http://example.com/news/article", expected: true},
		{name: "Valid News Link with Subdomain", href: "https://sub.example.com/news/article", expected: true},

		// Test cases where linkExists is true, but domainExists is false
		{name: "Invalid Link without HTTPS", href: "/news/article", expected: true},
		{name: "Invalid Link with HTTPS but without domain", href: "https:/news/article", expected: true},
		{name: "Invalid Link with HTTP but without domain", href: "http:/news/article", expected: true},

		// Test cases where domainExists is true, but linkExists is false
		{name: "Invalid Domain without News Path", href: "https://example.com/article", expected: false},
		{name: "Invalid Domain with Subpath", href: "https://example.com/subpath/article", expected: false},

		// Test cases where both linkExists and domainExists are false
		{name: "Invalid Link and Domain", href: "example.com/article", expected: false},
		{name: "Invalid Link and Domain (Empty String)", href: "", expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := linkFilter(test.href)
			if result != test.expected {
				t.Errorf("Expected %v, but got %v for input %s", test.expected, result, test.href)
			}
		})
	}
}

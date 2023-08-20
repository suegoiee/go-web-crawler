package main

import (
	"testing"
)

func TestLinkFilter(t *testing.T) {
	cases := []struct {
		name string
		href string
		want bool
	}{
		{name: "crorrect 1", href: "https://tw.stock.yahoo.com/news/q2%E5%95%86%E8%BE%A6%EF%BC%81a%E8%BE%A6%E7%A9%BA%E7%BD%AE%E7%8E%8744%EF%BC%85%EF%BC%8C%E7%A7%9F%E9%87%91qoq%E5%BE%AE%E5%8D%8706%EF%BC%85-064152206.html", want: true},
		{name: "wrong 1", href: "https://www.momoshop.com.tw/category/DgrpCategory.jsp?d_code=4300100677&p_orderType=4&showType=chessboardType&osm=googleKw&utm_source=google&utm_medium=cpc5-eCPC", want: false},
		{name: "wrong 2", href: "/living/378672", want: false},
		{name: "crorrect 2", href: "/news/living/378672", want: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := linkFilter(tt.href); got != tt.want {
				t.Errorf("%v is wrong", tt.name)
			}
		})
	}
}

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
		{name: "wrong 1", href: "https://tw.news.yahoo.com/%E6%96%B0%E9%9D%92%E5%AE%89%E6%88%BF%E8%B2%B8%E4%B8%8A%E8%B7%AF-%E6%88%BF%E5%B8%82%E5%B0%87%E5%87%BA%E7%8F%BE2%E8%AE%8A%E5%8C%96-%E4%BD%8E%E7%B8%BD%E5%83%B9%E6%88%90%E4%B8%BB%E6%B5%81-062900729.html", want: false},
		{name: "wrong 2", href: "/%E6%96%B0%E9%9D%92%E5%AE%89%E6%88%BF%E8%B2%B8%E4%B8%8A%E8%B7%AF-%E6%88%BF%E5%B8%82%E5%B0%87%E5%87%BA%E7%8F%BE2%E8%AE%8A%E5%8C%96-%E4%BD%8E%E7%B8%BD%E5%83%B9%E6%88%90%E4%B8%BB%E6%B5%81-062900729.html", want: false},
		{name: "crorrect 2", href: "/news/ai%E6%A6%82%E5%BF%B5%E8%82%A1%E9%A0%98%E6%BC%B2-%E5%8F%B0%E8%82%A1%E7%88%86%E9%87%8F4600%E5%84%84%E5%85%83%E4%B8%8A%E6%8F%9A99-37%E9%BB%9E-055328708.html", want: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := linkFilter(tt.href); got != tt.want {
				t.Errorf("%v is wrong", tt.name)
			}
		})
	}
}

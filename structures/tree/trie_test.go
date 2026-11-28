package tree

import (
	"testing"
)

func TestTrie(t *testing.T) {
	trie := NewTrie[string]()
	trie.Insert("/", "root")
	trie.Insert("/users", "users")
	trie.Insert("/users/:id", "user_detail")
	trie.Insert("/users/:id", "user_detail")
	trie.Insert("/users/:id/posts", "user_posts")
	trie.Insert("/files/*filename", "file_server")
	trie.Insert("/api/v1/:service/*", "api_handler")
	trie.Insert("/static/*filepath", "static")
	trie.Insert("/api/v1/posts", "posts")

	for _, pattern := range []string{
		"/",                           // 静态路径
		"/users",                      // 静态路径
		"/users/test_uer",             // 参数路径
		"/users/test_uer/posts",       // 参数路径
		"/files/userprofile",          // 通配符路径
		"/api/v1/tasks/tid1234567890", // 参数路径
		"/user/123",
		"/static/css/style.css",
		"/api/v1/posts",
	} {
		route, params, found := trie.Search(pattern)
		if !found {
			t.Logf("not found %s", pattern)
		} else {
			t.Logf("found %s %v", route, params)
		}
	}
}

// BenchmarkTrieInsert 测试插入性能
// cpu: Intel(R) Core(TM) i5-14600KF							static paths				20		 3817927	       310.9 ns/op	     280 B/op	      15 allocs/op
// cpu: Intel(R) Core(TM) i5-14600KF							dynamic paths				20		 4929130	       230.0 ns/op	     296 B/op	       9 allocs/op
// cpu: Intel(R) Core(TM) i5-14600KF							complex dynamic paths		20		    4640	      263776 ns/op	  154916 B/op	    3900 allocs/op
// cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		static paths				8		 2314869	       525.2 ns/op	     280 B/op	      15 allocs/op
// cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		dynamic paths				8		 3056307	       408.4 ns/op	     296 B/op	       9 allocs/op
// cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		complex dynamic paths		8		    2916	      404929 ns/op	  154937 B/op	    3901 allocs/op
//
//	func BenchmarkTrieInsert(b *testing.B) {
//		b.Run("简单路径插入", func(b *testing.B) {
//			trie := NewTrie[string]()
//			patterns := []string{"/", "/users", "/posts", "/comments", "/api/v1"}
//			b.ReportAllocs()
//			b.ResetTimer()
//
//			for i := 0; i < b.N; i++ {
//				for _, pattern := range patterns {
//					trie.Insert(pattern, "handler_"+pattern)
//				}
//			}
//		})
//
//		b.Run("带参数路径插入", func(b *testing.B) {
//			trie := NewTrie[string]()
//			patterns := []string{"/users/:id", "/posts/:postId", "/:lang/users/:id"}
//			b.ReportAllocs()
//			b.ResetTimer()
//
//			for i := 0; i < b.N; i++ {
//				for _, pattern := range patterns {
//					trie.Insert(pattern, "handler_"+pattern)
//				}
//			}
//		})
//
//		b.Run("大规模路径插入", func(b *testing.B) {
//			trie := NewTrie[int]()
//			b.ReportAllocs()
//			b.ResetTimer()
//
//			for i := 0; i < b.N; i++ {
//				// 每轮插入1000个路由
//				for j := 0; j < 1000; j++ {
//					pattern := "/api/v" + strconv.Itoa(j%10) + "/resource/" + strconv.Itoa(j)
//					trie.Insert(pattern, j)
//				}
//			}
//		})
//	}
//
// // BenchmarkTrieSearch 测试搜索性能
// // cpu: Intel(R) Core(TM) i5-14600KF						static paths				20		18065649	       63.88 ns/op	     104 B/op	       3 allocs/op
// // cpu: Intel(R) Core(TM) i5-14600KF						dynamic param paths			20		 9671016	       127.9 ns/op	     330 B/op	       4 allocs/op
// // cpu: Intel(R) Core(TM) i5-14600KF						dynamic wildcard paths		20		 7430827	       164.9 ns/op	     448 B/op	       5 allocs/op
// // cpu: Intel(R) Core(TM) i5-14600KF						not found paths				10		11812418	       95.67 ns/op	     213 B/op	       3 allocs/op
// //
// // cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		static paths				8		10413692	       117.4 ns/op	     104 B/op	       3 allocs/op
// // cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		dynamic param paths			8		 5726457	       211.4 ns/op	     330 B/op	       4 allocs/op
// // cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		dynamic wildcard paths		8		 4468950	       269.0 ns/op	     448 B/op	       5 allocs/op
// // cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		not found paths				8		 7396630	       161.6 ns/op	     213 B/op	       3 allocs/op
//
//	func BenchmarkTrieSearch(b *testing.B) {
//		// 准备测试数据
//		trie := prepareTestTrie(1000)
//
//		b.Run("静态路径搜索", func(b *testing.B) {
//			paths := []string{"/", "/users", "/api/v1", "/deep/nested/path"}
//			b.ReportAllocs()
//			b.ResetTimer()
//
//			for i := 0; i < b.N; i++ {
//				path := paths[i%len(paths)]
//				trie.Search(path)
//			}
//		})
//
//		b.Run("参数路径搜索", func(b *testing.B) {
//			paths := []string{"/users/123", "/posts/456", "/api/v1/resource/789"}
//			b.ReportAllocs()
//			b.ResetTimer()
//
//			for i := 0; i < b.N; i++ {
//				path := paths[i%len(paths)]
//				trie.Search(path)
//			}
//		})
//
//		b.Run("通配符路径搜索", func(b *testing.B) {
//			paths := []string{"/files/docs/report.pdf", "/static/css/main.css"}
//			b.ReportAllocs()
//			b.ResetTimer()
//
//			for i := 0; i < b.N; i++ {
//				path := paths[i%len(paths)]
//				trie.Search(path)
//			}
//		})
//
//		b.Run("未找到路径搜索", func(b *testing.B) {
//			paths := []string{"/notfound", "/invalid/path", "/api/v999/resource"}
//			b.ReportAllocs()
//			b.ResetTimer()
//
//			for i := 0; i < b.N; i++ {
//				path := paths[i%len(paths)]
//				trie.Search(path)
//			}
//		})
//	}
//
// cpu: Intel(R) Core(TM) i5-14600KF		20    	 3189684	       378.6 ns/op	     992 B/op	      12 allocs/op
func BenchmarkSearch(b *testing.B) {
	trie := NewTrie[string]()
	trie.Insert("/user/:id", "user")
	trie.Insert("/static/*filepath", "static")
	trie.Insert("/api/v1/posts", "posts")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		trie.Search("/user/123")
		trie.Search("/static/css/style.css")
		trie.Search("/api/v1/posts")
	}
}

//// BenchmarkTrieConcurrent 测试并发性能
//// cpu: Intel(R) Core(TM) i5-14600KF						8		 22675866	       51.58 ns/op	     340 B/op	       4 allocs/op
//// cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		8		 10900864	       112.6 ns/op	     340 B/op	       4 allocs/op
//func BenchmarkTrieConcurrent(b *testing.B) {
//	trie := prepareTestTrie(500)
//
//	b.Run("并发搜索", func(b *testing.B) {
//		b.ReportAllocs()
//		b.ResetTimer()
//
//		b.RunParallel(func(pb *testing.PB) {
//			paths := []string{"/users/123", "/api/v1/resource/456", "/static/file.txt", "/notfound"}
//			i := 0
//			for pb.Next() {
//				path := paths[i%len(paths)]
//				trie.Search(path)
//				i++
//			}
//		})
//	})
//}
//
//// 准备测试用的Trie树
//func prepareTestTrie(size int) *Trie[string] {
//	trie := NewTrie[string]()
//
//	// 插入静态路由
//	trie.Insert("/", "root")
//	trie.Insert("/users", "users")
//	trie.Insert("/api/v1", "api_v1")
//	trie.Insert("/deep/nested/path", "deep_path")
//
//	// 插入参数路由
//	for i := 0; i < size/2; i++ {
//		pattern := "/users/:id" + strconv.Itoa(i)
//		trie.Insert(pattern, "user_handler_"+strconv.Itoa(i))
//	}
//
//	// 插入通配符路由
//	trie.Insert("/files/*", "file_server")
//	trie.Insert("/static/*", "static_server")
//	trie.Insert("/api/*", "api_catch_all")
//
//	// 插入复杂模式
//	for i := 0; i < size/4; i++ {
//		pattern := "/api/v:version/resource/:id"
//		trie.Insert(pattern, "api_resource_"+strconv.Itoa(i))
//	}
//
//	return trie
//}
//
//// TestSplitPattern 测试路径分割函数
//func TestSplitPattern(t *testing.T) {
//	tests := []struct {
//		input    string
//		expected []string
//	}{
//		{"/", []string{}},
//		{"/users", []string{"users"}},
//		{"/users/:id", []string{"users", ":id"}},
//		{"/users/:id/posts", []string{"users", ":id", "posts"}},
//		{"users/:id/", []string{"users", ":id"}},
//		{"//users//:id//", []string{"users", ":id"}},
//		{"", []string{}},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.input, func(t *testing.T) {
//			result := SplitPattern(tt.input)
//			if len(result) != len(tt.expected) {
//				t.Errorf("SplitPattern(%s) = %v, want %v", tt.input, result, tt.expected)
//				return
//			}
//
//			for i := range result {
//				if result[i] != tt.expected[i] {
//					t.Errorf("SplitPattern(%s)[%d] = %v, want %v", tt.input, i, result[i], tt.expected[i])
//				}
//			}
//		})
//	}
//}

func TestTrie_Get(t *testing.T) {
	trie := NewTrie[string]()
	trie.Insert("/users/:id", "detail")
	trie.Insert("/files/*", "files")

	if v, ok := trie.Get("/users/:id"); !ok || v != "detail" {
		t.Fatalf("expected exact match for pattern, got %q, %v", v, ok)
	}

	// Get 只做完全匹配，不参与通配符匹配
	if _, ok := trie.Get("/users/123"); ok {
		t.Fatal("Get should not match wildcard patterns")
	}
	if _, ok := trie.Get("/missing"); ok {
		t.Fatal("Get should return false for unknown pattern")
	}
}

// TestTrie_CustomDelimMultiWildcard 验证自定义分隔符下多层通配符收集的剩余路径使用该分隔符
//   - 回归点：search 曾硬编码 "/" 拼接剩余路径，自定义分隔符时参数值被错误地改成 "/"
func TestTrie_CustomDelimMultiWildcard(t *testing.T) {
	trie := NewTrie[string]()
	trie.SetDelim(".")
	trie.Insert("a.b.*rest", "route")

	value, params, ok := trie.Search("a.b.c.d.e")
	if !ok {
		t.Fatal("expected match for a.b.c.d.e")
	}
	if value != "route" {
		t.Fatalf("value = %q, want route", value)
	}
	if got := params["rest"]; got != "c.d.e" {
		t.Fatalf("params[rest] = %q, want c.d.e", got)
	}

	// 精确查询同样遵循自定义分隔符
	if v, ok := trie.Get("a.b.*rest"); !ok || v != "route" {
		t.Fatalf("Get with custom delim = %q, %v", v, ok)
	}
}

// TestTrie_DefaultDelimMultiWildcard 确保默认分隔符行为未回归
func TestTrie_DefaultDelimMultiWildcard(t *testing.T) {
	trie := NewTrie[string]()
	trie.Insert("/static/*filepath", "static")

	_, params, ok := trie.Search("/static/css/style.css")
	if !ok {
		t.Fatal("expected match for /static/css/style.css")
	}
	if got := params["filepath"]; got != "css/style.css" {
		t.Fatalf("params[filepath] = %q, want css/style.css", got)
	}
}

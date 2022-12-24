package cache

import (
	"fmt"
	"testing"
	"time"
	// gCache "github.com/patrickmn/go-cache"
)

func TestLFUCache(t *testing.T) {
	cacheModule := New[string](CacheOpts{
		Strategy: LFU,
		Cap:      3,
	})

	cacheModule.Set("key_1", "value_1", 2000)
	cacheModule.Set("key_2", "value_2", 1)
	cacheModule.Set("key_3", "value_3", -1)
	cacheModule.Set("key_4", "value_4", 2000)

	fmt.Println(cacheModule.Get("key_0"))

	fmt.Println(cacheModule.Get("key_1"))

	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_4"))
	fmt.Println(cacheModule.Get("key_4"))
	fmt.Println(cacheModule.Get("key_4"))
	time.Sleep(time.Second * 1)
	fmt.Println(cacheModule.Get("key_2"))
	fmt.Println(cacheModule.Get("key_2"))
	fmt.Println(cacheModule.Get("key_2"))

	fmt.Println(cacheModule.Get("key_4"))
	fmt.Println(cacheModule.Get("key_4"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))
	fmt.Println(cacheModule.Get("key_3"))

	cacheModule.Set("key_5", "value_5", 2000)
	fmt.Println(cacheModule.Get("key_5"))

	cacheModule.Set("key_6", "value_6", 2000)

	fmt.Println("cap", cacheModule.cap)
	fmt.Println("leastF", cacheModule.leastF)

	cacheModule.freqMap.Range(func(key, value any) bool {
		fmt.Println("freq: ", key, "=", value)
		return true
	})

	cacheModule.values.Range(func(key, value any) bool {
		fmt.Println("key: ", key, "=", value)
		return true
	})

	// for i := 0; i < 4; i++ {
	// 	go cacheModule.Get("key_1")
	// 	go cacheModule.Get("key_2")
	// 	go cacheModule.Get("key_3")
	// 	go cacheModule.Get("key_4")
	// 	go cacheModule.Get("key_5")
	// }

	// for i := 0; i < 3; i++ {
	// 	go cacheModule.Get("key_6")
	// 	go cacheModule.Get("key_7")
	// 	go cacheModule.Get("key_8")
	// 	go cacheModule.Get("key_9")
	// 	go cacheModule.Get("key_10")
	// }

	// time.Sleep(time.Second)

	// cacheModule.freqMap.Range(func(key, value any) bool {
	// 	dll := value.(*DLL[string])

	// 	fmt.Println(
	// 		"frequency", key,
	// 		"dll size", dll.size,
	// 		"cap", cacheModule.cap,
	// 		"least freq", cacheModule.leastF,
	// 	)
	// 	fmt.Println(" - dll head", dll.head)
	// 	if dll.head != nil {
	// 		fmt.Println(" - dll head.nex", dll.head.next)
	// 	}
	// 	if dll.tail != nil {
	// 		fmt.Println(" - dll tail.prev", dll.tail.prev)
	// 	}
	// 	fmt.Println(" - dll tail", dll.tail)
	// 	return true
	// })
}

// frequency 4 dll size 5 cap 90 least freq 3
//  - dll head &{0xc0001704a0 <nil> value_1}
//  - dll head.nex &{0xc0001704c0 0xc000170480 value_2}
//  - dll tail.prev &{0xc000170500 0xc0001704c0 value_4}
//  - dll tail &{<nil> 0xc0001704e0 value_5}
// frequency 3 dll size 5 cap 90 least freq 3
//  - dll head &{0xc000170680 <nil> value_6}
//  - dll head.nex &{0xc0001706a0 0xc000170660 value_7}
//  - dll tail.prev &{0xc0001706e0 0xc0001706a0 value_9}
//  - dll tail &{<nil> 0xc0001706c0 value_10}

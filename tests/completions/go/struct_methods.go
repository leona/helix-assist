package main

type Counter struct {
	count int
}

func (c *Counter) Increment() {
	<CURSOR>
}

// Expected: completion should increment c.count

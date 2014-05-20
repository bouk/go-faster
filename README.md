# go-faster

## Installation

`go get github.com/bouk/go-faster`

## Basic usage

You can make any go program run faster by using more goroutines and channels! Example:

```go
# cat test.go
package main

func a(c int) int {
    c += 1
    return c
}

func main() {
    println(a(2))
}
# go-faster test.go | tee test.go
package main

func a(c int) chan int {
    result := make(chan int)
    go func() {
        result <- func() int {
            c += 1
            return c
        }()
    }()
    return result
}

func main() {
    println(<-a(2))
}
# go run test.go
3
```

Because the speed of a go program is measured in the amount of `go` statements and channels you can make any program faster by simply doing `go-faster <file.go>`.

## Even faster

There is no limitation on the number of times you can run `go-faster` on your program.

```go
# go-faster test.go | tee test.go
package main

func a(c int) chan chan chan int {
    result := make(chan chan chan int)
    go func() {
        result <- func() chan chan int {
            result := make(chan chan int)
            go func() {
                result <- func() chan int {
                    result := <-make(chan int)
                    go func() {
                        result <- func() int {
                            c += 1
                            return c
                        }()
                    }()
                    return result
                }()
            }()
            return result
        }()
    }()
    return result
}

func main() {
    println(<-<-<-a(2))
}
```

We hope that one day we can have infinite goroutines and channels.

# logs

example:

```
package main

import (
  "github.com/ti-net2/goutil/flag"
  "github.com/ti-net2/goutil/logs"
  "github.com/spf13/pflag"
)

func AddFlags(fs *pflag.FlagSet) {
  var demo string
  fs.IPVar(&demo, "demo", demo, ""+
		"add flags demo")
}


func main(){
  AddFlags(pflag.CommandLine)

  flag.InitFlags()
  logs.InitLogs()
  defer logs.FlushLogs()  
}

```

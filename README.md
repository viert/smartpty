## SmartPTY Golang library

**SmartPTY** is a simple expect/pexpect like library for go which makes you able to start a terminal with a standard **exec.Cmd** inside and react to certain expressions coming from the terminal output.

The most obvious way you can use it is starting a shell or a ssh session and make it type password for you automatically. You can find _sudo.go_ in _examples_ folder which does exactly that.

#### Methods

**func Create(\*exec.Cmd) \*SmartPTY**
creates a smartpty object based on the given Cmd

**func (sp \*SmartPTY) Start() error**
starts the terminal. Be sure to configure reaction callbacks before calling Start()

**func (sp \*SmartPTY) Close()**
closes the terminal and stops all the goroutines. This is not very useful as you can always call myCmd.Process.Kill() which will close related file descriptors and SmartPTY goroutines processing stdin/stdout should then stop automatically.

**func (sp \*SmartPTY) Once(expr \*regexp.Regexp, cb ExpressionCallback)**
**func (sp \*SmartPTY) Always(expr \*regexp.Regexp, cb ExpressionCallback)**
**func (sp \*SmartPTY) Times(expr \*regexp.Regexp, cb ExpressionCallback, int times)**
These functions create a reaction callback based on `expr` argument. When SmartPTY finds a chunk of data matching the expression, the `cb` function is called. The difference between these functions are kinda self-explanatory: `Once()` will run the callback just once, `Always()` will run its callback every time SmartPTY receives the matching chunk of data, and `Times()` will react exactly `times` times.

**type ExpressionCallback func(data []byte, tty \*os.File) []byte**
This is the callback function signature. SmartPTY will pass the whole data chunk matching the expression, the pseudo-terminal _\*os.File_ object where you can respond to using `tty.Write([]byte)`. In the end of each callback you can either return the whole chunk (_return data_) or modify it as you wish to, for example you may want to remove the _"Password:"_ prompt from the output.

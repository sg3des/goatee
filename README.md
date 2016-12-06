# GOATee - simple gtk2 text editor written on Go

# Configure

`goatee.conf` is example of config file, text editor tries to get it by `XDG_CONFIG_PATH/goatee/` or from working directory.

# Features

 * multiple homogeneous(*full width*) Tabs
 * auto detect charset
 * smart detect text file language
 * binary view(`hexdump -C`)

# Requirements
 
 * gtk2
 * gtksourceview2 
 * enca/libenca - *Charset analyser and converter*

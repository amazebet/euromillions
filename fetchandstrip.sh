#!/bin/bash
curl -s http://www.jogossantacasa.pt/web/SCHome/ | grep '<.*>[0-9].*</span>' -m 7 | sed -e 's/<.*>\(.*\)<\/.*>/\1/' -e 's/[ ]*//g' -e 's/[	]//g'

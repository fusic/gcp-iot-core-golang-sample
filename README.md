GCP Cloud IoT Core sample written in Golang

I've wrote some article too: https://qiita.com/take_cheeze/items/2b28eff1d7f1092ea1ac

## How to use
Build the code and run:
```
./gcp-iot-core-golang-sample \
    -project_id=${PROJECT_ID} \
    -registry_id=${REGISTRY_ID} \
    -device_id=${DEVICE_ID} \
    -algorithm=${RS256 or ES256} \
    -private_key_file=${PRIVATE_KEY_FILE_PATH}
```

## License
Copyright 2017 Takeshi Watanabe

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

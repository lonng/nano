# Nano -- golang based micro game server framework

## Chat Room Demo
implement a chat room in 100 lines with golang and websocket

- server
```go
package main

import (
	"net/http"
	"log"

	"github.com/lonnng/nano"
	"github.com/lonnng/nano/component"
	"github.com/lonnng/nano/serialize/json"
	"github.com/lonnng/nano/session"
)


type Room struct {
	component.Base
	group *nano.Group
}

type UserMessage struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type JoinResponse struct {
	Code   int    `json:"code"`
	Result string `json:"result"`
}

func NewRoom() *Room {
	return &Room{
		group: nano.NewGroup("room"),
	}
}

func (r *Room) Join(s *session.Session, msg []byte) error {
	s.Bind(s.ID)   // binding session uid
	r.group.Add(s) // add session to group
	return s.Response(JoinResponse{Result: "sucess"})
}

func (r *Room) Message(s *session.Session, msg *UserMessage) error {
	return r.group.Broadcast("onMessage", msg)
}

func main() {
	nano.Register(NewRoom())
	nano.SetSerializer(json.NewSerializer())

	log.SetFlags(log.LstdFlags | log.Llongfile)

	nano.SetCheckOriginFunc(func(_ *http.Request) bool { return true })
	nano.ListenWithOptions(":3250", true)
}
```
  
- client
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Chat Demo</title>
</head>
<body>
<div id="container">
    <ul>
        <li v-for="msg in messages">[<span style="color:red;">{{msg.name}}</span>]{{msg.content}}</li>
    </ul>
    <div class="controls">
        <input type="text" v-model="nickname">
        <input type="text" v-model="inputMessage">
        <input type="button" v-on:click="sendMessage" value="Send">
    </div>
</div>
<script src="http://cdnjs.cloudflare.com/ajax/libs/vue/1.0.26/vue.min.js" type="text/javascript"></script>
<!--[starx websocket library](https://github.com/lonnng/nano-client-websocket)-->
<script src="protocol.js" type="text/javascript"></script>
<script src="starx-wsclient.js" type="text/javascript"></script>
<script>
    var v = new Vue({
        el: "#container",
        data: {
            nickname:'guest' + Date.now(),
            inputMessage:'',
            messages: []
        },
        methods: {
            sendMessage: function () {
                console.log(this.inputMessage);
                starx.notify('Room.Message', {name: this.nickname, content: this.inputMessage});
                this.inputMessage = '';
            }
        }
    });

    var onMessage = function (msg) {
        v.messages.push(msg)
    };

    var join = function (data) {
        console.log(data);
        if(data.code === 0) {
            v.messages.push({name:'system', content:data.result});
            starx.on('onMessage', onMessage)
        }
    };

    starx.init({host: '127.0.0.1', port: 3250}, function () {
        console.log("initialized")
        starx.request("Room.Join", {}, join);
    })
</script>
</body>
</html>
```

## The MIT License

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.

# Route compression

***STATUS: DRAFT***

## Why need route compression

In practice, network bandwidth is a worthwhile consideration. Especially for mobile clients,
the network resource is often not very rich, in order to save network resources, it is often
needed to increase the effective payload ratio.

Using the chat application as an example, when a user send a chat message, the route information
is required, as shown below:

```javascript

nano.request('Room.Join',
  //...
);

```
The routing information indicates that the request should be handled by send method of Join on
Room component. When server pushing messages to the client, route also should be specified to
indicate a handler. In the chat example, there are onAdd, onLeave and other routes. Considering
if a chat message is very short such as just a letter, but when being sent, it should be added
a complete routing information, this would result in a very low effective payload ratio and wasting
network resource. The direct idea to solve this problem is to shorten the routing information.
On the server side, routing information is fixed while the server is determined. On the client
side, although you can use a very short name for route, but it may be unreadable.

## How to implement

To address this situation, nano provides the dictionary-based route compression.

* For the server side, nano scans all route information;
* For the client side, the developer needs routes map.

Then, nano would get all the routes of client side and server side and then maps each route to
a small integer. Currently nano route compression supports limit. The implementation of current
stage is that fetching routes when handshake phase, if you enable route compression, then the
client and server will synchronize the dictionary in handshake phase while establishing a
connection, so that the small integer can be used to replace the route later, and it can reduce
the transmission cost.

## Summary

So far, The format of transmission message between client and server is json. Indeed, while json
is very convenient, but it also brought some redundant information, which can be ommited to reduce
transmission cost.

***Copyright***:Parts of above content and figures come from [Pomelo Route compression](https://github.com/NetEase/pomelo/wiki/Route-compression)

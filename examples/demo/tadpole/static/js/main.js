var settings = new Settings();

var debug = false;
var isStatsOn = false;
var uiEnabled = false;

var app;
var runLoop = function () {
    app.update();
    app.draw();
};

var showUILayer = function () {
    uiEnabled = true;
    $('#ui').show();
};

var hideUILayer = function () {
    uiEnabled = false;
    $('#ui').hide();
};

var initApp = function () {
    if (app != null) {
        return;
    }
    var canvas = document.getElementById('canvas');
    app = new App(settings, canvas);

    window.addEventListener('resize', app.resize, false);
    canvas.addEventListener('mousemove', app.mousemove, false);
    canvas.addEventListener('mousedown', app.mousedown, false);
    canvas.addEventListener('mouseup', app.mouseup, false);

    canvas.addEventListener('touchstart', app.touchstart, false);
    canvas.addEventListener('touchend', app.touchend, false);
    canvas.addEventListener('touchcancel', app.touchend, false);
    canvas.addEventListener('touchmove', app.touchmove, false);

    canvas.addEventListener('keydown', app.keydown, false);
    canvas.addEventListener('keyup', app.keyup, false);
    canvas.onselectstart = function () {
        return false;
    }

    setInterval(runLoop, 30);
};

var forceInit = function () {
    initApp();
    document.getElementById('unsupported-browser').style.display = "none";
    return false;
};

if (Modernizr.canvas && Modernizr.websockets) {
    initApp();
} else {
    document.getElementById('unsupported-browser').style.display = "block";
    document.getElementById('force-init-button').addEventListener('click', forceInit, false);
}

var addStats = function () {
    if (isStatsOn) {
        return;
    }
    // Draw fps
    var stats = new Stats();
    document.getElementById('fps').appendChild(stats.domElement);

    setInterval(function () {
        stats.update();
    }, 1000 / 60);

    // Array Remove - By John Resig (MIT Licensed)
    Array.remove = function (array, from, to) {
        var rest = array.slice((to || from) + 1 || array.length);
        array.length = from < 0 ? array.length + from : from;
        return array.push.apply(array, rest);
    };
    isStatsOn = true;
};

//document.addEventListener('keydown',function(e) {
//	if(e.which == 27) {
//		addStats();
//	}
//})

if (debug) {
    addStats();
}

$(function () {
    $('a[rel=external]').click(function (e) {
        e.preventDefault();
        window.open($(this).attr('href'));
    });
});

$('#message-close').click(function () {
    console.log('sdfsdf')
});
// change Vuejs delimiters
Vue.config.delimiters = ['<%', '%>'];

var messageDialog = new Vue({
    el: '#message-dialog',
    data: {
        message: {
            avatar: 'http://pic.qiushibaike.com/system/avtnew/3118/31183960/thumb/20160216132422.jpg',
            content: "hello world"
        }
    },
    methods: {
        drop: function (event) {
            console.log('drop');
            hideUILayer()
        },
        try: function (event) {
            $('#message-dialog').hide();
            $('#message-list').show();
        }
    }
});

var messageList = new Vue({
    el: '#message-list',
    data: {
        name: 'zhangsan',
        currentMessage: '',
        messages: [
            {
                location:'left',
                content: 'hello world'
            }
        ]
    },
    methods: {
        sendMessage: function () {
            this.messages.push({
                location:'right',
                content: this.currentMessage
            })
        },
        back: function () {
            $('#message-list').hide();
            $('#conversation-list').show();
        },
        showDetail: function () {
            $('#user-profile').show()
        }
    }
});

var conversations = new Vue({
    el: "#conversation-list",
    data: {
        conversations:[
            {
                avatar: 'http://pic.qiushibaike.com/system/avtnew/3118/31183960/thumb/20160216132422.jpg',
                name: 'hello',
                message: 'this is test text',
                time: '10:20'
            },
            {
                avatar: 'http://pic.qiushibaike.com/system/avtnew/3118/31183960/thumb/20160216132422.jpg',
                name: 'hello',
                message: 'this is test text',
                time: '10:20'
            },
            {
                avatar: 'http://pic.qiushibaike.com/system/avtnew/3118/31183960/thumb/20160216132422.jpg',
                name: 'hello',
                message: 'this is test text',
                time: '10:20'
            },
            {
                avatar: 'http://pic.qiushibaike.com/system/avtnew/3118/31183960/thumb/20160216132422.jpg',
                name: 'hello',
                message: 'this is test text',
                time: '10:20'
            }
        ]
    },
    methods: {
        selectConversation: function (index) {
            console.log(index);
            var c = this.conversations[index];
            $('#message-list').show();
            $('#conversation-list').hide();
        }
    }
});

new Vue({
    el: '#user-profile',
    data:{profile:{
        avatar: 'http://pic.qiushibaike.com/system/avtnew/3118/31183960/thumb/20160216132422.jpg',
        location: 'China',
        name: 'Chris Lonng',
        sex: 'Male',
        signature: 'hello world, i am here'
    }},
    methods:{}
});
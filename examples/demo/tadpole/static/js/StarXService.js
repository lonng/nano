var StarXService = function (model) {
    var self = this;
    this.hasConnection = false;

    this.sendUpdate = function (data) {
        starx.notify("World.Update", {
            id: data.id,
            x: data.x,
            y: data.y,
            name: data.name,
            momentum: data.momentum,
            angle: data.angle
        });
    };

    this.sendMessage = function(msg){
        starx.notify("World.Message", {
            message: msg
        });
    };

    var welcomeHandler = function (data) {
        self.hasConnection = true;
        model.userTadpole.id = data.id;
        model.userTadpole.name = "GUEST" + data.id;
        model.tadpoles[data.id] = model.tadpoles[-1];
        delete model.tadpoles[-1];

        $('#chat').initChat();
    };

    var joyLogin = function () {
        starx.request("World.Enter", {}, welcomeHandler)
    };

    var updateHandler = function (data) {
        var newtp = false;

        if (!model.tadpoles[data.id]) {
            newtp = true;
            model.tadpoles[data.id] = new Tadpole();
            model.arrows[data.id] = new Arrow(model.tadpoles[data.id], model.camera);
        }

        var tadpole = model.tadpoles[data.id];

        if (tadpole.id == model.userTadpole.id) {
            tadpole.name = data.name;
            return;
        } else {
            tadpole.name = data.name;
        }

        if (newtp) {
            tadpole.x = data.x;
            tadpole.y = data.y;
        } else {
            tadpole.targetX = data.x;
            tadpole.targetY = data.y;
        }

        tadpole.angle = data.angle;
        tadpole.momentum = data.momentum;

        tadpole.timeSinceLastServerUpdate = 0;
    };

    var leaveHandler = function (data) {
        if (!!model.tadpoles[data.id]) {
            delete model.tadpoles[data.id];
            delete model.arrows[data.id];
        }
    };

    var closeHandler = function (data) {
        webSocketService.hasConnection = false;
        $('#cant-connect').fadeIn(300);
    };

    var messageHandler = function (data) {
        var tadpole = model.tadpoles[data.id];
        if (!tadpole) {
            return;
        }
        tadpole.timeSinceLastServerUpdate = 0;
        tadpole.messages.push(new Message(data.message));
    };

    var wsInit = function () {
        starx.request("Manager.Login", {username: "123123123", cipher: "12312312312"}, joyLogin);
        starx.on("close", closeHandler);
        starx.on("update", updateHandler);
        starx.on("leave", leaveHandler);
        starx.on("message", messageHandler)
    };

    // Constructor
    (function () {
        starx.init({host: model.settings.host, port: model.settings.port}, wsInit)
    })();
};
App = function (aSettings, aCanvas) {
    var app = this;

    var model,
        canvas,
        context,
        starXService;
    mouse = {x: 0, y: 0, worldx: 0, worldy: 0, tadpole: null},
        keyNav = {x: 0, y: 0},
        messageQuota = 5;

    app.update = function () {
        if (messageQuota < 5 && model.userTadpole.age % 50 == 0) {
            messageQuota++;
        }

        // Update usertadpole
        if (keyNav.x != 0 || keyNav.y != 0) {
            model.userTadpole.userUpdate(model.tadpoles, model.userTadpole.x + keyNav.x, model.userTadpole.y + keyNav.y);
        }
        else {
            var mvp = getMouseWorldPosition();
            mouse.worldx = mvp.x;
            mouse.worldy = mvp.y;
            model.userTadpole.userUpdate(model.tadpoles, mouse.worldx, mouse.worldy);
        }

        if (model.userTadpole.age % 6 == 0 && model.userTadpole.changed > 1 && starXService.hasConnection) {
            model.userTadpole.changed = 0;
            starXService.sendUpdate(model.userTadpole);
        }

        model.camera.update(model);

        // Update tadpoles
        for (id in model.tadpoles) {
            model.tadpoles[id].update(mouse);
        }

        // Update waterParticles
        for (i in model.waterParticles) {
            model.waterParticles[i].update(model.camera.getOuterBounds(), model.camera.zoom);
        }

        // Update arrows
        for (i in model.arrows) {
            var cameraBounds = model.camera.getBounds();
            var arrow = model.arrows[i];
            arrow.update();
        }
    };


    app.draw = function () {
        model.camera.setupContext();

        // Draw waterParticles
        for (i in model.waterParticles) {
            model.waterParticles[i].draw(context);
        }

        // Draw tadpoles
        for (id in model.tadpoles) {
            model.tadpoles[id].draw(context);
        }

        // Start UI layer (reset transform matrix)
        model.camera.startUILayer();

        // Draw arrows
        for (i in model.arrows) {
            model.arrows[i].draw(context, canvas);
        }
    };

    app.mousedown = function (e) {
        mouse.clicking = true;

        if (mouse.tadpole && mouse.tadpole.hover && mouse.tadpole.onclick(e)) {
            return;
        }
        if (model.userTadpole && e.which == 1) {
            model.userTadpole.momentum = model.userTadpole.targetMomentum = model.userTadpole.maxMomentum;
        }
    };

    app.mouseup = function (e) {
        if (model.userTadpole && e.which == 1) {
            model.userTadpole.targetMomentum = 0;
        }
    };

    app.mousemove = function (e) {
        mouse.x = e.clientX;
        mouse.y = e.clientY;
    };

    app.keydown = function (e) {
        if (e.keyCode == keys.up) {
            keyNav.y = -1;
            model.userTadpole.momentum = model.userTadpole.targetMomentum = model.userTadpole.maxMomentum;
            e.preventDefault();
        }
        else if (e.keyCode == keys.down) {
            keyNav.y = 1;
            model.userTadpole.momentum = model.userTadpole.targetMomentum = model.userTadpole.maxMomentum;
            e.preventDefault();
        }
        else if (e.keyCode == keys.left) {
            keyNav.x = -1;
            model.userTadpole.momentum = model.userTadpole.targetMomentum = model.userTadpole.maxMomentum;
            e.preventDefault();
        }
        else if (e.keyCode == keys.right) {
            keyNav.x = 1;
            model.userTadpole.momentum = model.userTadpole.targetMomentum = model.userTadpole.maxMomentum;
            e.preventDefault();
        }
    };
    app.keyup = function (e) {
        if (e.keyCode == keys.up || e.keyCode == keys.down) {
            keyNav.y = 0;
            if (keyNav.x == 0 && keyNav.y == 0) {
                model.userTadpole.targetMomentum = 0;
            }
            e.preventDefault();
        }
        else if (e.keyCode == keys.left || e.keyCode == keys.right) {
            keyNav.x = 0;
            if (keyNav.x == 0 && keyNav.y == 0) {
                model.userTadpole.targetMomentum = 0;
            }
            e.preventDefault();
        }
    };

    app.touchstart = function (e) {
        e.preventDefault();
        mouse.clicking = true;

        if (model.userTadpole) {
            model.userTadpole.momentum = model.userTadpole.targetMomentum = model.userTadpole.maxMomentum;
        }

        var touch = e.changedTouches.item(0);
        if (touch) {
            mouse.x = touch.clientX;
            mouse.y = touch.clientY;
        }
    };
    app.touchend = function (e) {
        if (model.userTadpole) {
            model.userTadpole.targetMomentum = 0;
        }
    };
    app.touchmove = function (e) {
        e.preventDefault();

        var touch = e.changedTouches.item(0);
        if (touch) {
            mouse.x = touch.clientX;
            mouse.y = touch.clientY;
        }
    };

    app.resize = function (e) {
        resizeCanvas();
    };

    app.sendMessage = function (msg) {
        starXService.sendMessage(msg)
    };

    var getMouseWorldPosition = function () {
        return {
            x: (mouse.x + (model.camera.x * model.camera.zoom - canvas.width / 2)) / model.camera.zoom,
            y: (mouse.y + (model.camera.y * model.camera.zoom - canvas.height / 2)) / model.camera.zoom
        }
    };

    var resizeCanvas = function () {
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight;
    };

    // Constructor
    (function () {
        canvas = aCanvas;
        context = canvas.getContext('2d');
        resizeCanvas();

        model = new Model();
        model.settings = aSettings;

        model.userTadpole = new Tadpole();
        model.userTadpole.id = -1;
        model.tadpoles[model.userTadpole.id] = model.userTadpole;

        model.waterParticles = [];
        for (var i = 0; i < 150; i++) {
            model.waterParticles.push(new WaterParticle());
        }

        model.camera = new Camera(canvas, context, model.userTadpole.x, model.userTadpole.y);
        model.arrows = {};

        starXService = new StarXService(model);
    })();
};

var WaterParticle = function() {
    var self = this;
    self.x = 0;
    self.y = 0;
    self.z = Math.random() + 0.3;
    self.size = 1.2;
    self.opacity = Math.random() * 0.8 + 0.1;
    self.update = function(bounds) {
        if (self.x == 0 || self.y == 0) {
            self.x = Math.random() * (bounds[1].x - bounds[0].x) + bounds[0].x;
            self.y = Math.random() * (bounds[1].y - bounds[0].y) + bounds[0].y;
        }
        // Wrap around screen
        self.x = self.x < bounds[0].x ? bounds[1].x : self.x;
        self.y = self.y < bounds[0].y ? bounds[1].y : self.y;
        self.x = self.x > bounds[1].x ? bounds[0].x : self.x;
        self.y = self.y > bounds[1].y ? bounds[0].y : self.y;
    }
    ;
    self.draw = function(context) {
        // Draw circle
        context.fillStyle = 'rgba(226,219,226,' + self.opacity + ')';
        //context.fillStyle = '#fff';
        context.beginPath();
        context.arc(self.x, self.y, self.z * self.size, 0, Math.PI * 2, true);
        context.closePath();
        context.fill();
    }
    ;
}
;
var Message = function(msg) {
    var message = this;
    this.age = 1;
    this.maxAge = 300;
    this.message = msg;
    this.update = function() {
        this.age++;
    }
    ;
    this.draw = function(context, x, y, i) {
        var fontsize = 8;
        context.font = fontsize + "px 'proxima-nova-1','proxima-nova-2', arial, sans-serif";
        context.textBaseline = 'hanging';
        var paddingH = 3;
        var paddingW = 6;
        var messageBox = {
            width: context.measureText(message.message).width + paddingW * 2,
            height: fontsize + paddingH * 2,
            x: x,
            y: (y - i * (fontsize + paddingH * 2 + 1)) - 20
        };
        var fadeDuration = 20;
        var opacity = (message.maxAge - message.age) / fadeDuration;
        opacity = opacity < 1 ? opacity : 1;
        context.fillStyle = 'rgba(255,255,255,' + opacity / 20 + ')';
        drawRoundedRectangle(context, messageBox.x, messageBox.y, messageBox.width, messageBox.height, 10);
        context.fillStyle = 'rgba(255,255,255,' + opacity + ')';
        context.fillText(message.message, messageBox.x + paddingW, messageBox.y + paddingH, 100);
    }
    ;
    var drawRoundedRectangle = function(ctx, x, y, w, h, r) {
        var r = r / 2;
        ctx.beginPath();
        ctx.moveTo(x, y + r);
        ctx.lineTo(x, y + h - r);
        ctx.quadraticCurveTo(x, y + h, x + r, y + h);
        ctx.lineTo(x + w - r, y + h);
        ctx.quadraticCurveTo(x + w, y + h, x + w, y + h - r);
        ctx.lineTo(x + w, y + r);
        ctx.quadraticCurveTo(x + w, y, x + w - r, y);
        ctx.lineTo(x + r, y);
        ctx.quadraticCurveTo(x, y, x, y + r);
        ctx.closePath();
        ctx.fill();
    }
}
;
var Arrow = function(tadpole, camera) {
    var arrow = this;
    this.x = 0;
    this.y = 0;
    this.tadpole = tadpole;
    this.camera = camera;
    this.angle = 0;
    this.distance = 10;
    this.opacity = 1;
    this.update = function() {
        arrow.angle = Math.atan2(tadpole.y - arrow.camera.y, tadpole.x - arrow.camera.x);
    }
    ;
    this.draw = function(context, canvas) {
        var cameraBounds = arrow.camera.getBounds();
        if (arrow.tadpole.x < cameraBounds[0].x || arrow.tadpole.y < cameraBounds[0].y || arrow.tadpole.x > cameraBounds[1].x || arrow.tadpole.y > cameraBounds[1].y) {
            var size = 4;
            var arrowDistance = 100;
            var angle = arrow.angle;
            var w = (canvas.width / 2) - 10;
            var h = (canvas.height / 2) - 10;
            var aa = Math.atan(h / w);
            var ss = Math.cos(angle);
            var cc = Math.sin(angle);
            if ((Math.abs(angle) + aa) % Math.PI / 2 < aa) {
                arrowDistance = w / Math.abs(ss);
            } else {
                arrowDistance = h / Math.abs(cc);
            }
            var x = (canvas.width / 2) + Math.cos(arrow.angle) * arrowDistance;
            var y = (canvas.height / 2) + Math.sin(arrow.angle) * arrowDistance;
            var point = calcPoint(x, y, this.angle, 2, size);
            var side1 = calcPoint(x, y, this.angle, 1.5, size);
            var side2 = calcPoint(x, y, this.angle, 0.5, size);
            // Draw arrow
            context.fillStyle = 'rgba(255,255,255,' + arrow.opacity + ')';
            context.beginPath();
            context.moveTo(point.x, point.y);
            context.lineTo(side1.x, side1.y);
            context.lineTo(side2.x, side2.y);
            context.closePath();
            context.fill();
        }
    }
    ;
    var calcPoint = function(x, y, angle, angleMultiplier, length) {
        return {
            x: x + Math.cos(angle + Math.PI * angleMultiplier) * length,
            y: y + Math.sin(angle + Math.PI * angleMultiplier) * length
        }
    }
    ;
}
;
var TadpoleTail = function(tadpole) {
    var tail = this;
    tail.joints = [];
    var tadpole = tadpole;
    var jointSpacing = 1.4;
    var animationRate = 0;
    tail.update = function() {
        animationRate += (.2 + tadpole.momentum / 10);
        for (var i = 0, len = tail.joints.length; i < len; i++) {
            var tailJoint = tail.joints[i];
            var parentJoint = tail.joints[i - 1] || tadpole;
            var anglediff = (parentJoint.angle - tailJoint.angle);
            while (anglediff < -Math.PI) {
                anglediff += Math.PI * 2;
            }
            while (anglediff > Math.PI) {
                anglediff -= Math.PI * 2;
            }
            tailJoint.angle += anglediff * (jointSpacing * 3 + (Math.min(tadpole.momentum / 2, Math.PI * 1.8))) / 8;
            tailJoint.angle += Math.cos(animationRate - (i / 3)) * ((tadpole.momentum + .3) / 40);
            if (i == 0) {
                tailJoint.x = parentJoint.x + Math.cos(tailJoint.angle + Math.PI) * 5;
                tailJoint.y = parentJoint.y + Math.sin(tailJoint.angle + Math.PI) * 5;
            } else {
                tailJoint.x = parentJoint.x + Math.cos(tailJoint.angle + Math.PI) * jointSpacing;
                tailJoint.y = parentJoint.y + Math.sin(tailJoint.angle + Math.PI) * jointSpacing;
            }
        }
    }
    ;
    tail.draw = function(context) {
        var path = [[], []];
        for (var i = 0, len = tail.joints.length; i < len; i++) {
            var tailJoint = tail.joints[i];
            var falloff = (tail.joints.length - i) / tail.joints.length;
            var jointSize = (tadpole.size - 1.8) * falloff;
            var x1 = tailJoint.x + Math.cos(tailJoint.angle + Math.PI * 1.5) * jointSize;
            var y1 = tailJoint.y + Math.sin(tailJoint.angle + Math.PI * 1.5) * jointSize;
            var x2 = tailJoint.x + Math.cos(tailJoint.angle + Math.PI / 2) * jointSize;
            var y2 = tailJoint.y + Math.sin(tailJoint.angle + Math.PI / 2) * jointSize;
            path[0].push({
                x: x1,
                y: y1
            });
            path[1].push({
                x: x2,
                y: y2
            });
        }
        for (var i = 0; i < path[0].length; i++) {
            context.lineTo(path[0][i].x, path[0][i].y);
        }
        path[1].reverse();
        for (var i = 0; i < path[1].length; i++) {
            context.lineTo(path[1][i].x, path[1][i].y);
        }
    }
    ;
    (function() {
        for (var i = 0; i < 15; i++) {
            tail.joints.push({
                x: 0,
                y: 0,
                angle: Math.PI * 2,
            })
        }
    })();
}
;
var Tadpole = function() {
    var tadpole = this;
    this.x = Math.random() * 300 - 150;
    this.y = Math.random() * 300 - 150;
    this.size = 4;
    this.name = '';
    this.age = 0;
    this.hover = false;
    this.momentum = 0;
    this.maxMomentum = 3;
    this.angle = Math.PI * 2;
    this.targetX = 0;
    this.targetY = 0;
    this.targetMomentum = 0;
    this.messages = [];
    this.timeSinceLastActivity = 0;
    this.changed = 0;
    this.timeSinceLastServerUpdate = 0;

    this.update = function(mouse) {
        if (uiEnabled){
            tadpole.momentum = 0;
            tadpole.targetMomentum = 0;
        }
        tadpole.timeSinceLastServerUpdate++;
        tadpole.x += Math.cos(tadpole.angle) * tadpole.momentum;
        tadpole.y += Math.sin(tadpole.angle) * tadpole.momentum;
        if (tadpole.targetX != 0 || tadpole.targetY != 0) {
            tadpole.x += (tadpole.targetX - tadpole.x) / 20;
            tadpole.y += (tadpole.targetY - tadpole.y) / 20;
        }
        // Update messages
        for (var i = tadpole.messages.length - 1; i >= 0; i--) {
            var msg = tadpole.messages[i];
            msg.update();
            if (msg.age == msg.maxAge) {
                tadpole.messages.splice(i, 1);
            }
        }
        // Update tadpole hover/mouse state
        if (Math.sqrt(Math.pow(tadpole.x - mouse.worldx, 2) + Math.pow(tadpole.y - mouse.worldy, 2)) < tadpole.size + 2) {
            tadpole.hover = true;
            mouse.tadpole = tadpole;
        } else {
            if (mouse.tadpole && mouse.tadpole.id == tadpole.id) {//mouse.tadpole = null;
            }
            tadpole.hover = false;
        }
        tadpole.tail.update();
    }
    ;
    this.onclick = function(e) {
        if (e.ctrlKey && e.which == 1) {
            if (isAuthorized() && tadpole.hover) {
                window.open("http://twitter.com/" + tadpole.name.substring(1));
                return true;
            }
        } else if (e.which == 2) {
            //todo:open menu
            e.preventDefault();
            return true;
        }
        return false;
    }
    ;
    this.userUpdate = function(tadpoles, angleTargetX, angleTargetY) {
        this.age++;
        var prevState = {
            angle: tadpole.angle,
            momentum: tadpole.momentum
        };
        // Angle to targetx and targety (mouse position)
        var anglediff = ((Math.atan2(angleTargetY - tadpole.y, angleTargetX - tadpole.x)) - tadpole.angle);
        while (anglediff < -Math.PI) {
            anglediff += Math.PI * 2;
        }
        while (anglediff > Math.PI) {
            anglediff -= Math.PI * 2;
        }
        tadpole.angle += anglediff / 5;
        // Momentum to targetmomentum
        if (tadpole.targetMomentum != tadpole.momentum) {
            tadpole.momentum += (tadpole.targetMomentum - tadpole.momentum) / 20;
        }
        if (tadpole.momentum < 0) {
            tadpole.momentum = 0;
        }
        tadpole.changed += Math.abs((prevState.angle - tadpole.angle) * 3) + tadpole.momentum;
        if (tadpole.changed > 1) {
            this.timeSinceLastServerUpdate = 0;
        }
    }
    ;

    this.draw = function(context) {
        var opacity = Math.max(Math.min(20 / Math.max(tadpole.timeSinceLastServerUpdate - 300, 1), 1), .2).toFixed(3);
        if (tadpole.hover && isAuthorized()) {
            context.fillStyle = 'rgba(192, 253, 247,' + opacity + ')';
            // context.shadowColor   = 'rgba(249, 136, 119, '+opacity*0.7+')';
        } else {
            context.fillStyle = 'rgba(226,219,226,' + opacity + ')';
        }
        context.shadowOffsetX = 0;
        context.shadowOffsetY = 0;
        context.shadowBlur = 6;
        context.shadowColor = 'rgba(255, 255, 255, ' + opacity * 0.7 + ')';
        // Draw circle
        context.beginPath();
        context.arc(tadpole.x, tadpole.y, tadpole.size, tadpole.angle + Math.PI * 2.7, tadpole.angle + Math.PI * 1.3, true);
        tadpole.tail.draw(context);
        context.closePath();
        context.fill();
        context.shadowBlur = 0;
        context.shadowColor = '';
        drawName(context);
        drawMessages(context);
    }
    ;
    var isAuthorized = function() {
        return tadpole.name.charAt('0') == "@";
    }
    ;
    var drawName = function(context) {
        var opacity = Math.max(Math.min(20 / Math.max(tadpole.timeSinceLastServerUpdate - 300, 1), 1), .2).toFixed(3);
        context.fillStyle = 'rgba(226,219,226,' + opacity + ')';
        context.font = 7 + "px 'proxima-nova-1','proxima-nova-2', arial, sans-serif";
        context.textBaseline = 'hanging';
        var width = context.measureText(tadpole.name).width;
        context.fillText(tadpole.name, tadpole.x - width / 2, tadpole.y + 8);
    }
    ;
    var drawMessages = function(context) {
        tadpole.messages.reverse();
        for (var i = 0, len = tadpole.messages.length; i < len; i++) {
            tadpole.messages[i].draw(context, tadpole.x + 10, tadpole.y + 5, i);
        }
        tadpole.messages.reverse();
    }
    ;
    // Constructor
    (function() {
        tadpole.tail = new TadpoleTail(tadpole);
    })();
}
;

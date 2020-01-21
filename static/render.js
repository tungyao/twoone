// 新版浏览器可以使用
class Render {
    constructor(node) {
        this.node = null;
        this.name= null;
        this.status= null;
        this.setNode(node);
        this.status = 0;
        this._prefix = 0;
        this._node = document.createElement("div");
    }

    _findPrefix(node, data) {
        let c = node;
        this._findAttr(c, data);
        for (let i of c.childNodes) {
            this._findAttr(i, data);
            if (i.nodeType === 3) {
                i = this._parseDom(i, data)
            }
            if (i.nodeType === 1) {
                this._findChild(i.childNodes, data)
            }
        }
        return c
    }

    _findAttr(node, data) {
        if (typeof node.attributes === "undefined") {
            return;
        }
        for (let i of node.attributes) {
            this._parseDom(i, data)
        }
    }

    _findChild(node, data) {
        this._findAttr(node, data);
        if (typeof node.length === "undefined") {
            if (typeof node === "object" && node.nodeType === 1) {
                this._parseDom(node, data);
            }
            return;
        }
        for (let i of node) {
            this._findAttr(i, data);
            if (i.nodeType === 3) {
                this._parseDom(i, data)
            } else if (i.nodeType === 1) {
                this._findChild(i.childNodes, data)
            }
        }
    }

    _parseDom(node, data) {
        let html = node.nodeValue;
        if (node.nodeValue === null) {
            html = node.innerHTML;
        }
        let a = 0;
        for (let j in html) {
            if (html[j] === "{") {
                a = j;
            }
            if (html[j] === "}") {
                let ns = html.substr(a, j - a + 1);
                let nss = html.substr(a, j - a);
                let d = nss.split(".");
                let key;
                if (d.length > 2) {
                    key = this._fff(data[d[1]], d.pop());
                } else {
                    key = data[d[1]];
                }
                let value = html.replace(ns, key);
                if (node.nodeValue === null) {
                    node.innerHTML = value;
                } else {
                    node.nodeValue = value;
                }
                html = value;
            }
        }
        return node;
    };

    _fff(object, key) {
        if (typeof object === 'object') {
            for (let i in object) {
                if (typeof object[i] === 'object') {
                    return this._fff(object[i], key);
                }
                if (key === i) {
                    return object[key];
                }
            }
        }
        return object[key];
    }

    setNode(node) {
        this.node = document.querySelectorAll(node);
        if (typeof this.node !== "object") {
            this.node = document.getElementsByClassName(node.substr(1, node.length - 1));
        }
        this.name = node;
        this._check();
        return this;
    }

    _check() {
        for (let i of this.node) {
            if (i.attributes.getNamedItem("render-if") !== null) {
                if (i.attributes.getNamedItem("render-if").value === "false") {
                    i.style.display = "none";
                }
            }
        }
    }

    if(status) {
        this.setNode(this.name);
        for (let i of this.node) {
            if (status) {
                if (!i.attributes.getNamedItem("status")) {
                    i.style.removeProperty("display")
                }
            } else {
                i.style.display = "none";
            }
        }
        return this;
    }

    _same(nodes, data, type) {
        let node = nodes;
        let dname;
        if (type === 0) {
            dname = nodes.attributes.getNamedItem("render-for").value;
        } else if (type === 1) {
            dname = nodes.attributes.getNamedItem("render-obj").value;
        }
        if (node.style.display === "none") {
            node.style.removeProperty("display");
            // node.attributes.removeNamedItem("render-for");
        }
        let n = node.nextSibling;
        while (n) {
            let nn = n;
            n = n.nextSibling;
            nn.remove();
        }
        if (node.attributes.getNamedItem("status") && this.status !== 0) {
            node.attributes.removeNamedItem("status");
        }
        if (type === 0) {
            for (let j of data[dname]) {
                nodes.after(this._findPrefix(node.cloneNode(true), j));
            }
        } else if (type === 1) {
            nodes.after(this._findPrefix(node.cloneNode(true), data[dname]));
        }
        nodes.style.display = "none";
        nodes.attributes.setNamedItem(document.createAttribute("status"));
    }

    obj(data) {
        this._same(this.node[0], data, 1)
    }

    for(data) {
        for (let i of this.node) {
            this._same(i, data, 0);
        }
        this.status += 1;
        return this;
    }

    append(node) {
        let dom  = document.createElement("div");
        dom.innerHTML =node;
        for (let i of this.node) {
           i.appendChild(dom)
        }
        return this;
    }
}

// 旧版浏览器使用 主要针对不兼容after的浏览器 把class删去
var oRender = function oRender() {
};
oRender.prototype = {
    _findPrefix: function (node, data) {
        var c = node;
        this._findAttr(c, data);
        for (let i of c.childNodes) {
            this._findAttr(i, data);
            if (i.nodeType === 3) {
                i = this._parseDom(i, data)
            }
            if (i.nodeType === 1) {
                this._findChild(i.childNodes, data)
            }
        }
        return c
    },
    _findAttr: function (node, data) {
        if (typeof node.attributes === "undefined") {
            return;
        }
        for (let i of node.attributes) {
            this._parseDom(i, data)
        }
    },

    _findChild: function (node, data) {
        this._findAttr(node, data);
        if (typeof node.length === "undefined") {
            if (typeof node === "object" && node.nodeType === 1) {
                this._parseDom(node, data);
            }
            return;
        }
        for (let i of node) {
            this._findAttr(i, data);
            if (i.nodeType === 3) {
                this._parseDom(i, data)
            } else if (i.nodeType === 1) {
                this._findChild(i.childNodes, data)
            }
        }
    },

    _parseDom: function (node, data) {
        var html = node.nodeValue;
        if (node.nodeValue === null) {
            html = node.innerHTML;
        }
        var a = 0;
        for (var j in html) {
            if (html[j] === "{") {
                a = j;
            }
            if (html[j] === "}") {
                var ns = html.substr(a, j - a + 1);
                var nss = html.substr(a, j - a);
                var d = nss.split(".");
                var key;
                if (d.length > 2) {
                    key = this._fff(data[d[1]], d.pop());
                } else {
                    key = data[d[1]];
                }
                var value = html.replace(ns, key);
                if (node.nodeValue === null) {
                    node.innerHTML = value;
                } else {
                    node.nodeValue = value;
                }
                html = value;
            }
        }
        return node;
    },

    _fff: function (object, key) {
        if (typeof object === 'object') {
            for (var i in object) {
                if (typeof object[i] === 'object') {
                    return this._fff(object[i], key);
                }
                if (key === i) {
                    return object[key];
                }
            }
        }
        return object[key];
    },

    setNode: function (node) {
        this.node = document.querySelectorAll(node);
        if (typeof this.node !== "object") {
            this.node = document.getElementsByClassName(node.substr(1, node.length - 1));
        }
        this.name = node;
        this._check();
        return this;
    },

    _check: function () {
        for (var i of this.node) {
            if (i.attributes.getNamedItem("render-if") !== null) {
                if (i.attributes.getNamedItem("render-if").value === "false") {
                    i.style.display = "none";
                }
            }
        }
    },

    if(status) {
        this.setNode(this.name);
        for (var i of this.node) {
            if (status) {
                if (!i.attributes.getNamedItem("status")) {
                    i.style.removeProperty("display")
                }
            } else {
                i.style.display = "none";
            }
        }
        return this;
    },

    _same(nodes, data, type) {
        var node = nodes;
        var dname;
        if (type === 0) {
            dname = nodes.attributes.getNamedItem("render-for").value;
        } else if (type === 1) {
            dname = nodes.attributes.getNamedItem("render-obj").value;
        }
        if (node.style.display === "none") {
            node.style.removeProperty("display");
            // node.attributes.removeNamedItem("render-for");
        }
        var n = node.nextSibling;
        while (n) {
            let nn = n;
            n = n.nextSibling;
            nn.remove();
        }
        if (node.attributes.getNamedItem("status") && this.status !== 0) {
            node.attributes.removeNamedItem("status");
        }
        if (type === 0) {
            for (let j of data[dname]) {
                var parent = nodes.parentNode;
                if (parent.lastChild === nodes) {
                    parent.appendChild(this._findPrefix(node.cloneNode(true), j));
                } else {
                    parent.insertBefore(this._findPrefix(node.cloneNode(true), j), nodes.nextSibling)
                }
            }
        } else if (type === 1) {
            var parent = nodes.parentNode;
            if (parent.lastChild === nodes) {
                parent.appendChild(this._findPrefix(node.cloneNode(true), data[dname]));
            } else {
                parent.insertBefore(this._findPrefix(node.cloneNode(true), data[dname]), nodes.nextSibling)
            }
        }
        nodes.style.display = "none";
        nodes.attributes.setNamedItem(document.createAttribute("status"));
    },

    obj: function (data) {
        this._same(this.node[0], data, 1)
    },

    for: function (data) {
        for (var i of this.node) {
            this._same(i, data, 0);
        }
        this.status += 1;
        return this;
    },

    append: function (node) {
        for (var i of this.node) {
            i.innerHTML = node
        }
        return this;
    }
};
function OldRender (){
    return  new oRender();
}
function NewRender(node) {
    return new Render().setNode(node);
}

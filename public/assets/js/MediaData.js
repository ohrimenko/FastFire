/**
 * MediaData - A JavaScript lib For Web Artisans
 * https://github.com/ohrimenko/mediastrim
 *
 * @package  mediastrim
 * @author  ohrimenko <ohrimenko.dmitro@gmail.com>
 *
 * Copyright 2022 ohrimenko
 * Released under the MIT license
 */

'use strict';

function MediaData(media, sources) {
    var $this = this;
    
    this.media = media;
    this.sources = sources;
    
    for (var i in this.sources) {
        if (typeof this.sources[i].manifest == "string") {
            this.sources[i].manifest = MediaData.ParseManifest(this.sources[i].manifest);
        }
    }
    
    this.sources.sort(function (a, b) {
        if (a.manifest.width < b.manifest.width) {
            return 1;
        }
        if (a.manifest.width > b.manifest.width) {
            return -1;
        }
        
        return 0;
    });
    
    this.initSkey();
    
    window.addEventListener('resize', function () {
        $this.initSkey();
    });
    
    this.currentTime = 0;
    this.timeLoadNext = 0;
    
    this.isInit = false;
    
    this.countInitMediaSource = 0;
    
    this.xhrs = {};
    this.xhrskey = 0;
    
    this.xhrfetchs = {};
    this.xhrfetchskey = 0;
    
    this.preLoadRanges = [];
    this.dataRanges = {};
    
    this.initMediaSource();
}

MediaData.startMediaSource = true; // Create MediaSource when object is created
MediaData.ajaxType = "fetch"; // fetch or xhr ajax request type
MediaData.partTimeInterval = 1; // Grouping the input manifest by 1 second
MediaData.loadTimeInterval = 15; // We load for 15 seconds.
MediaData.loadTimeWaiting = 5; // We load the first 5 seconds by 1 second.
MediaData.partRange = 3; // We cut into three parts the buffer loading for the first 5 seconds if the input manifest contains data with a frequency of more than a second.
MediaData.loadSeekTime = 2000; // We start the video two seconds after rewinding if the rewinding of the video is not completed and does not change.
MediaData.stopOtherVideo = true; // When playing a video, stop others.
MediaData.preLoadTime = 0.8; // We store 0.8 second of video in cache for the effect of instant video playback when you click on play.
MediaData.urlPreLoad = ""; // Link to load the buffer cache with one request for all videos at once.

MediaData.TimerId = null;
MediaData.mediStreams = {};

MediaData.userAgent = "";
MediaData.userBrowser = "";

if (window.navigator && window.navigator.userAgent) {
    MediaData.userAgent = window.navigator.userAgent.toLowerCase();
}

if (/firefox/.test(MediaData.userAgent)) {
    MediaData.userBrowser = "firefox";
} else if (/chrome/.test(MediaData.userAgent)) {
    MediaData.userBrowser = "chrome";
} else if (/safari/.test(MediaData.userAgent)) {
    MediaData.userBrowser = "safari";
} else if (/opera/.test(MediaData.userAgent)) {
    MediaData.userBrowser = "opera";
} else if((/mozilla/.test(MediaData.userAgent) && !/firefox/.test(MediaData.userAgent) && !/chrome/.test(MediaData.userAgent) && !/safari/.test(MediaData.userAgent) && !/opera/.test(MediaData.userAgent)) || /msie/.test(MediaData.userAgent)) {
    MediaData.userBrowser = "ie";
}

if (MediaData.userBrowser == "firefox") {
    // Firefox has a problem with the seeked event. It doesn't always work.
    // This parameter helps solve this problem by intentionally firing this event.
    MediaData.loadSeekTime = 500;
}

MediaData.ParseManifest = function (dataManifest) {
    var manifest = {};
    
    manifest.type = "";
    manifest.size = 0;
    manifest.width = 0;
    manifest.height = 0;
    manifest.duration = 0;
    manifest.init = [];
    manifest.media = [];
    
    var data = dataManifest.split("\n");
    var row;
    
    var cntEmpty = 0;
    
    for (var i in data) {
        data[i] = data[i].trim();
        
        if (cntEmpty >= 2) {
            row = data[i].split(",", 2);
            
            if (row.length == 2 && row[0] && row[1]) {
                manifest.media.push([parseInt(row[0]), parseFloat(row[1])]);
            }
        } else {
            if (data[i]) {
                row = data[i].split(":", 2);
                
                if (row.length == 2) {
                    switch (row[0]) {
                        case "type":
                            manifest.type = row[1];
                            break;
                        case "size":
                            manifest.size = parseInt(row[1]);
                            break;
                        case "width":
                            manifest.width = parseInt(row[1]);
                            break;
                        case "height":
                            manifest.height = parseInt(row[1]);
                            break;
                        case "duration":
                            manifest.duration = parseFloat(row[1]);
                            break;
                        case "init":
                            var init = row[1].split(",", 2);
                            
                            if (init.length == 2) {
                                manifest.init = [parseInt(init[0]), parseInt(init[1])];
                            }
                            break;
                    }
                }
            } else {
                cntEmpty++;
            }
        }
    }
    
    return manifest;
}

MediaData.fgets = function (dataManifest) {
}

MediaData.TimerInterval = function () {
    if (MediaData.TimerId) {
        clearInterval(MediaData.TimerId);
    }
    
    MediaData.TimerId = setInterval(function () {
        for (var i in MediaData.mediStreams) {
            if (MediaData.mediStreams[i].isPlay && MediaData.mediStreams[i].currentSegment < MediaData.mediStreams[i].segments.length) {
                MediaData.mediStreams[i].timerCb();
            }
        }
    }, 100);
};

MediaData.TimerInterval();

MediaData.prototype.initSkey = function() {
    this.minSKey = 0;
    this.maxSKey = 0;
    
    for (var i in this.sources) {
        this.sources[this.minSKey]['default'] = false;
        this.maxSKey = i;
    }
    
    for (var i in this.sources) {
        this.minSKey = i;
        
        if (this.sources[i].manifest.width < this.media.clientWidth) {
            break;
        }
    }
    
    this.sources[this.minSKey]['default'] = true;
    this.sKey = this.minSKey;
};

MediaData.prototype.initMediaSource = function() {
    if (!this.media.getAttribute("id")) {
        this.media.setAttribute("id", MediaData.randomString(20));
    }
    
    this.countInitMediaSource++;
    
    if (this.countInitMediaSource > 100) {
        return;
    }
    
    var $this = this;
    
    for (var i in this.xhrs) {
        if (this.xhrs[i].xhr) {
            this.xhrs[i].xhr.abort();
        }
    }
    
    for (var i in this.xhrfetchs) {
        if (this.xhrfetchs[i]) {
            this.xhrfetchs[i].ctr.abort();
        }
    }
    
    this.xhrs = {};
    
    this.xhrfetchs = {};
    
    if (this.mediaSource) {
        if (this.mediaSource.readyState === 'open') {
            this.sourceBuffer.abort();
            this.mediaSource.endOfStream();
        }
        
        if (typeof this.sourceBuffer == "object") {
            try {
                this.mediaSource.removeSourceBuffer(this.sourceBuffer);
            } catch (error) {
                console.error(error);
            }
        }
    }
    
    this.mediaSource = null;
    this.sourceBuffer = null;
    this.sKey = 0;
    this.sKeyInit = null;
    this.segments = [];
    this.loads = {};
    this.iload = 0;
    this.chunks = {};
    this.currentSegment = 0;
    this.lMimeCodec = null;
    this.canplay = false;
    this.isFetchInit = false;
    this.isPlay = false;
    this.timeLoadNext = 0;
    this.lastCurrentTime = 0;
    this.lastPlayTime = 0;
    this.msNoPlay = 0;
    this.isSeekSet = false;  
    this.timeWaiting = 0;  
    this.seek = false;
    this.timeSeeking = 0;
    this.countFetch = 0;
    
    if (this.media && this.sources.length > 0) {
        this.media.objMediaData = this;
        
        for (var i in this.sources) {
            if (this.sources[i]['default']) {
                this.sKey = i;
            }
        }
        
        if (!this.sources[this.sKey]) {
            for (var i in this.sources) {
                this.sKey = i;
                break;
            }
        }
        
        var lasttimecode = 0;
        var segment = [
            false,
            [],
            0,
        ];
        
        if (this.sources[this.sKey]) {
            for (var i in this.sources[this.sKey]['manifest']['media']) {
                if (
                    i > 0 && 
                    this.sources[this.sKey]['manifest']['media'][parseFloat(i)+1] && 
                    parseFloat(this.sources[this.sKey]['manifest']['media'][parseFloat(i)+1][1]) - lasttimecode > MediaData.partTimeInterval
                ) {
                    this.segments.push(segment);
                    segment = [
                        false,
                        [],
                        0,
                    ];
                    
                    lasttimecode = parseFloat(this.sources[this.sKey]['manifest']['media'][i][1]);
                    segment[2] = lasttimecode;
                }
                
                segment[1].push(i);   
            }
            
            this.segments.push(segment);
        }
        
        //console.log(this.segments);
        
        if ('MediaSource' in window) {
            if (MediaSource.isTypeSupported(this.getMimeCodec())) {
                if (this.currentTime > 0) {
                    this.isSeekSet = true; 
                }    
                
                if (MediaData.startMediaSource) {
                    $this.ctrlMediaSource();
                }
                
                this.media.addEventListener('click', function(e) {
                    $this.ctrlMediaSource(function () {
                        if (!$this.isFetchInit) {
                            setTimeout(function () {
                                $this.media.play();
                            }, 10);
                        }
                    });
                });
                
                this.media.addEventListener('play', function(e) {
                    if (MediaData.stopOtherVideo) {
                        for (var i in MediaData.mediStreams) {
                            if (MediaData.mediStreams[i] === $this) {
                                continue;
                            }
                        
                            MediaData.mediStreams[i].stop(true);
                        }
                    }
                    
                    if (!$this.mediaSource) {
                        $this.ctrlMediaSource(function () {
                            $this.fetchNextRange(); 
                            
                            $this.isPlay = true;
                            
                            $this.fetchInit();
                            
                            $this.play();
                        });
                    } else {
                        $this.isPlay = true;
            
                        $this.fetchInit();
            
                        $this.play();
                    }
                });
            } else {
                console.error('Unsupported MIME type or codec: ', this.getMimeCodec());
                
                this.media.src = this.sources[this.sKey]['url'];
            }
        } else {
            console.error('MediaSource: undefined');
            
            this.media.src = this.sources[this.sKey]['url'];
        }
    }
    
    MediaData.mediStreams[this.media.getAttribute("id")] = this;
    
    var currentSegment = this.currentSegment;
    
    if (this.sources[this.sKey]['manifest']['media'].length > 0 && this.segments[currentSegment]) {
        var ranges = [];
        
        var range = this.fetchNextRange(currentSegment, false, true);
        
        if (range && range.length > 0) {
            this.preLoadRanges.push(range);
            
            if (this.currentTime > 0) {
                for (var i in this.segments) {
                    if (parseFloat(this.segments[i][2]) > parseFloat(this.currentTime)) {
                        break;
                    }
                    currentSegment = i;
                }
            }
            
            if (this.getTimePreLoadRanges() < MediaData.preLoadTime || this.currentTime > 0) {
                for (var i = 0; i < 50; i++) {
                    currentSegment++;
                    
                    var range = this.fetchNextRange(currentSegment, true, true);    
                    
                    if (range && range.length > 0) {
                        this.preLoadRanges.push(range);
                    } else {
                        break;
                    }             
                    
                    if (this.getTimePreLoadRanges() >= MediaData.preLoadTime) {
                        break;
                    }
                }
            }
        }
    }
    
    this.countFetch = 0;
    
    //console.log(this.preLoadRanges);
};

MediaData.prototype.getTimePreLoadRanges = function() {
    var time = 0;
    
    for (var i in this.preLoadRanges) {
        for (var j in this.preLoadRanges[i]) {
            time = parseFloat(time) + (parseFloat(this.preLoadRanges[i][j]['timeInterval'][1]) - parseFloat(this.preLoadRanges[i][j]['timeInterval'][0]));
        }
    }
    
    return time;
};

MediaData.preLoad = function(cb) {
    if (MediaData.urlPreLoad) {
        var items = [];
        
        for (var m in MediaData.mediStreams) {
            for (var i in MediaData.mediStreams[m].preLoadRanges) {
                for (var j in MediaData.mediStreams[m].preLoadRanges[i]) {
                    MediaData.mediStreams[m].dataRanges[MediaData.mediStreams[m].preLoadRanges[i][j]['start']+"-"+MediaData.mediStreams[m].preLoadRanges[i][j]['end']] = {
                        status: "pending",
                        data: null,
                        segment: MediaData.mediStreams[m].preLoadRanges[i][j]                        
                    };
                    
                    items.push({
                        key: m,
                        url: MediaData.mediStreams[m].preLoadRanges[i][j].url,
                        range: [
                            MediaData.mediStreams[m].preLoadRanges[i][j]['start'],
                            MediaData.mediStreams[m].preLoadRanges[i][j]['end']
                        ],
                    });
                }
            }
        }
        
        //console.log("items", items);
        
        if (items.length) {
            if (MediaData.ajaxType == 'fetch' && 'fetch' in window) {
                fetch(MediaData.urlPreLoad, {
                    method: 'POST',
                    headers: {
                        'Content-type': 'application/x-www-form-urlencoded'
                    },
                    body: MediaData.httpBuildQuery({items: items}),
                }).then((response) => {
                        if (response.status >= 200 && response.status < 300) {
                            return response.arrayBuffer();
                        } else {
                            let error = new Error(response.statusText);
                            error.response = response;
                            throw error;
                        }
                }).then((response) => {
                    if (response) {
                        MediaData.responsePreLoad(items, response);
                    }
                    
                    if (cb) {
                        cb();
                    }
                }).catch((e) => {
                    if (cb) {
                        cb();
                    }
                });
            } else {
                var xhrLoad = new XMLHttpRequest;
                                
                xhrLoad.open('post', MediaData.urlPreLoad, true);
                xhrLoad.responseType = 'arraybuffer';
                xhrLoad.setRequestHeader('Content-type', 'application/x-www-form-urlencoded');
                xhrLoad.addEventListener('load', function () {
                    if (xhrLoad.response) {
                        MediaData.responsePreLoad(items, xhrLoad.response);
                    }
                    
                    if (cb) {
                        cb();
                    }
                });
                xhrLoad.addEventListener('error', function () {
                    if (cb) {
                        cb();
                    }
                });
                xhrLoad.addEventListener('abort', function () {
                    if (cb) {
                        cb();
                    }
                });
                
                xhrLoad.send(MediaData.httpBuildQuery({items: items}));
            }
        } else {
            if (cb) {
                cb();
            }
        }
        
        return;
    }
    
    for (var i in MediaData.mediStreams) {
        if (MediaData.mediStreams[i].preLoadComplete) {
            continue;
        }
        
        if (MediaData.mediStreams[i].preLoad(function () {
            MediaData.preLoad(cb);
        })) {
            return;
        }
    }
    
    if (cb) {
        cb();
    }
};

MediaData.responsePreLoad = function(items, response) {
    response = new Uint8Array(response);
    
    var length = 0;
    
    for (var i in items) {
        length += parseInt(items[i]["range"][1]) - parseInt(items[i]["range"][0]) + 1;
    }
        
    //console.log("responsePreLoad/items:", items.length, "/length:", length, "/response:", response.length);
    
    if (length == response.length) {
        for (var i in items) {
            var $this = MediaData.mediStreams[items[i]['key']];
            
            $this.dataRanges[items[i]['range'][0]+"-"+items[i]['range'][1]].status = "success";
            $this.dataRanges[items[i]['range'][0]+"-"+items[i]['range'][1]].data = response.slice(parseInt(items[i]["range"][0]), parseInt(items[i]["range"][1]) + 1);
        
            //console.log("Range Length:",$this.dataRanges[items[i]['range'][0]+"-"+items[i]['range'][1]].data.length);
            
            //console.log($this.dataRanges[items[i]['range'][0]+"-"+items[i]['range'][1]]);
            
            if ($this.mediaSource && $this.dataRanges[items[i]['range'][0]+"-"+items[i]['range'][1]].segment) {
                $this.fetchRange(
                    [$this.dataRanges[items[i]['range'][0]+"-"+items[i]['range'][1]].segment],
                    function (iSegment, mimeCodec, chunk) {
                        $this.addChuck(iSegment, mimeCodec, chunk);
                    }, function (status) {
                    }, 
                    true
                );
            }
        }
    }
};

MediaData.prototype.preLoad = function(cb) {
    var $this = this;
    
    this.preLoad;
    
    for (var i in this.preLoadRanges) {
        for (var j in this.preLoadRanges[i]) {
            var keyInterval = this.preLoadRanges[i][j]['start']+"-"+this.preLoadRanges[i][j]['end'];
            
            if (!this.dataRanges[keyInterval]) {
                this.dataRanges[keyInterval] = {
                    status: "new",
                    data: null,
                    segment: this.preLoadRanges[i][j],
                };
            }
                        
            if (this.dataRanges[keyInterval].status == "new") {
                this.dataRanges[keyInterval].status = "pending";
                
                if (MediaData.ajaxType == 'fetch' && 'fetch' in window) {
                    this.xhrfetchskey++;
        
                    var xhrfetchskey = this.xhrfetchskey;
            
                    this.xhrfetchs[xhrfetchskey] = {
                        ctr: new AbortController()
                    };
            
                    fetch(this.preLoadRanges[i][j].url+(this.preLoadRanges[i][j].url.indexOf("?")>0?"&":"?")+'bytes='+this.preLoadRanges[i][j].start+'-'+this.preLoadRanges[i][j].end, {
                        method: 'GET',
           	            headers: {
           	                'Range': 'bytes=' + this.preLoadRanges[i][j].start + '-' + this.preLoadRanges[i][j].end,
                        },
                        signal: this.xhrfetchs[xhrfetchskey].ctr.signal,
                    }).then((response) => {
                        if (response.status >= 200 && response.status < 300) {
                            return response.arrayBuffer();
                        } else {
                            let error = new Error(response.statusText);
                            error.response = response;
                            throw error;
                        }
                    }).then((response) => {
                        $this.dataRanges[keyInterval].status = "load";
                    
                        if (response) {
                            $this.dataRanges[keyInterval].status = "success";
                            $this.dataRanges[keyInterval].data = response;
                        }
                    
                        if ($this.xhrfetchs[xhrfetchskey]) {
                            delete $this.xhrfetchs[xhrfetchskey];
                        }
                        $this.preLoad(cb);
                    }).catch((e) => {
                        if ($this.xhrfetchs[xhrfetchskey]) {
                            delete $this.xhrfetchs[xhrfetchskey];
                        }
                        $this.preLoad(cb);
                    });
                } else {
                    this.xhrskey++;
                    var xhrskey = this.xhrskey;
                
                    var xhrLoad = new XMLHttpRequest;
                
                    this.xhrs[xhrskey] = {
                        xhr: xhrLoad
                    };
                    
                    xhrLoad.open('get', this.preLoadRanges[i][j].url+(this.preLoadRanges[i][j].url.indexOf("?")>0?"&":"?")+'bytes='+this.preLoadRanges[i][j].start+'-'+this.preLoadRanges[i][j].end, true);
                    xhrLoad.responseType = 'arraybuffer';
                    xhrLoad.setRequestHeader('Range', 'bytes=' + this.preLoadRanges[i][j].start + '-' + this.preLoadRanges[i][j].end);
                    xhrLoad.addEventListener('load', function () {
                        $this.dataRanges[keyInterval].status = "load";
                    
                        if (xhrLoad.response) {
                            $this.dataRanges[keyInterval].status = "success";
                            $this.dataRanges[keyInterval].data = xhrLoad.response;
                        }
                    
                        if ($this.xhrs[xhrskey]) {
                            delete $this.xhrs[xhrskey];
                        }
                        $this.preLoad(cb);
                    });
                    xhrLoad.addEventListener('error', function () {
                        $this.dataRanges[keyInterval].status = "error";
                        if ($this.xhrs[xhrskey]) {
                            delete $this.xhrs[xhrskey];
                        }
                        $this.preLoad(cb);
                    });
                    xhrLoad.addEventListener('abort', function () {
                        $this.dataRanges[keyInterval].status = "abort";
                        if ($this.xhrs[xhrskey]) {
                            delete $this.xhrs[xhrskey];
                        }
                        $this.preLoad(cb);
                    });
            
                    xhrLoad.send();
                }
                
                return true;
            }
        }
    }
    
    this.preLoadComplete = true;
    
    if (cb) {
        cb();
    }
    
    return false;
};

MediaData.prototype.start = function(load) {
    this.ctrlMediaSource();
    
    if (load) {
        this.checkBuffer(); 
    }
};

MediaData.prototype.ctrlMediaSource = function(cb) {
    if (!this.mediaSource) {
        this.newMediaSource(cb);        
    } else {
        if (cb) {
            cb();
        }
    }
};

MediaData.prototype.newMediaSource = function(cb) {
    var $this = this;
    
    this.mediaSource = new MediaSource;
                       
    this.media.src = URL.createObjectURL(this.mediaSource);
    this.mediaSource.addEventListener('sourceopen', function (_) {
        $this.sourceOpen(_);
        
        if (cb) {
            cb();
        }
    });
};

MediaData.prototype.sKeyPlus = function() {
    //alert(this.minSKey+"/"+this.maxSKey+"/"+this.sKey);
    
    if (this.sKey < this.maxSKey) {
        this.sKey++;
    }
};

MediaData.prototype.sKeyMinus = function() {
    if (this.sKey > this.minSKey) {
        this.sKey--;
    }
};

MediaData.prototype.reinit = function(cb) {
    var $this = this;
    
    this.setStartCurrentTime = true;
    
    this.initMediaSource();
    
    this.ctrlMediaSource(cb);
    
    console.log("reinit");
    
    setTimeout(function () {
        $this.play();
    }, 10);
    
};

MediaData.prototype.getMimeCodec = function() {
    if (this.sources[this.sKey]) {
        return this.sources[this.sKey]['manifest']['type'];
    }
    
    return "";
}

MediaData.prototype.sourceOpen = function(_) {
    var $this = this;
    
    this.sourceBuffer = this.mediaSource.addSourceBuffer(this.getMimeCodec());
    
    this.lMimeCodec = this.getMimeCodec();
    
    this.sourceBuffer.mode = "sequence";
    
    if (this.currentTime > 0) {
        this.media.currentTime = this.currentTime;
    }
    
    this.init();
};

MediaData.prototype.init = function() {
    var $this = this;
    
    if (!this.isInit && this.sources[this.sKey]) {
        this.media.addEventListener('canplay', function () {
            $this.canplay = true;
        });
        this.media.addEventListener('seeking', function(e) {
            $this.seeking(e);              
        });
        this.media.addEventListener('seeked', function(e) {
            $this.seeked(e);
        });
        this.media.addEventListener('error', function(e) {
            //$this.reinit();
        });
        this.media.addEventListener('pause', function(e) {
            $this.isPlay = false;
            
            $this.msNoPlay = 0;
            
            $this.fetchInit();
        });
        
        this.isInit = true;
    }
};

MediaData.prototype.timerCb = function() {
    if (this.seek || this.media.seeking) {
        return;
    }
    
    this.load();
    
    this.playControl();
};

MediaData.prototype.playControl = function(action, msNoPlay) {
    var time = Date.now();
    
    if (!msNoPlay) {
        msNoPlay = 2000;
    }
    
    if (this.lastCurrentTime != this.media.currentTime) {
        this.lastCurrentTime = this.media.currentTime;
        this.lastPlayTime = time;
        
        this.msNoPlay = 0;
    } else {
        this.msNoPlay = time - this.lastPlayTime;
    }
        
    if (this.isPlay && this.canplay && !this.isRePlay) {
        if (this.msNoPlay > msNoPlay) {
            for (var i in this.loads) {
                return;
            }
            
            //console.log("msNoPlay: ", msNoPlay);
            
            this.rePlay(action, msNoPlay);
        }
    }
};

MediaData.prototype.rePlay = function(action, msNoPlay) {
    var $this = this;
    
    if (!msNoPlay) {
        msNoPlay = 2000;
    }
    
    if (this.isPlay && this.msNoPlay > msNoPlay) {
        switch (action) {
            case "checkBuffer1":
                this.checkBuffer();
                
                setTimeout(function () {
                    $this.rePlay("checkBuffer2");
                }, 250);
                
                break;
            case "checkBuffer2":
                this.checkBuffer();
                
                setTimeout(function () {
                    $this.rePlay("checkBuffer3");
                }, 250);
                
                break;
            case "checkBuffer3":
                this.checkBuffer();
                
                setTimeout(function () {
                    $this.rePlay("reinit");
                }, 250);
                
                break;
            case "reinit":
                if ($this.msNoPlay > 3000) {
                    for (var i in $this.loads) {
                        return;
                    }
                }
                
                this.reinit(function () {
                    setTimeout(function () {
                        $this.isRePlay = false;
                        $this.media.play();
                        
                        setTimeout(function () {
                            $this.playControl();
                        }, 100);
                    }, 10);
                });
                
                break;
            default:
                this.isRePlay = true;
            
                //$this.rePlay("reinit");
                //return;
                
                this.checkBuffer();
                
                this.media.pause();
                this.media.play();
                
                //console.log("reinit: ", msNoPlay);
                
                setTimeout(function () {
                    $this.rePlay("reinit");
                }, 250);
            
                break;
        }
    } else {
        $this.isRePlay = false;
        this.msNoPlay = 0;                
    }
};

MediaData.prototype.fetchInit = function() {
    var $this = this;
    
    if (!this.isFetchInit && this.sources[this.sKey]) {
        //this.fetchNextRange();
        
        this.media.addEventListener('timeupdate', function(_) {
            if ($this.canplay) {
                $this.currentTime = $this.media.currentTime;
            }
            
            if (!$this.media.seeking) {
                $this.checkBuffer(_);
            }
        });
        
        this.media.addEventListener('waiting', function(_) {
            //console.log("waiting");
                
            $this.checkBuffer(_);
            $this.sKeyPlus();
                
            //$this.timeWaiting = $this.media.currentTime;                
        });
        
        this.isFetchInit = true;
    }
};

MediaData.prototype.play = function (currentTime) {
    var $this = this;
    
    if (!this.mediaSource) {
        if (currentTime > 0) {
            this.setStartCurrentTime = true;
        }
        
        $this.ctrlMediaSource(function () {
            $this.fetchNextRange();
            $this.play(currentTime);
        });   
        return;  
    }   
    
    $this.isPlay = true;
    
    if (currentTime >= 0) {
        $this.media.currentTime = $this.currentTime = currentTime;
        
        $this.isSeekSet = true;     
    }
    
    if (!$this.isFetchInit) {
        $this.fetchInit();
    }
    
    if ($this.canplay) {
        if ($this.media.paused) {
            $this.media.play();
        }
    } else {
        setTimeout(function () {
            $this.play();
        }, 10);
    }
};

MediaData.prototype.stop = function (abort) {
    this.isPlay = false;
    
    this.msNoPlay = 0;
    
    if (!this.media.paused) {
        this.media.pause();
    }
    
    if (abort) {
        for (var i in this.xhrs) {
            if (this.xhrs[i].range) {
                continue;
            }
            
            if (this.xhrs[i].xhr) {
                this.xhrs[i].xhr.abort();
            }
            
            delete this.xhrs[i];
        }
        
        for (var i in this.xhrfetchs) {
            if (this.xhrfetchs[i].range) {
                continue;
            }
            
            if (this.xhrfetchs[i]) {
                this.xhrfetchs[i].ctr.abort();
            }
            
            delete this.xhrfetchs[i];
        }
        
        if (this.sourceBuffer && this.mediaSource.readyState == 'open') {
            //this.sourceBuffer.abort();
        }
    }
};

MediaData.prototype.getFileLength = function (url, cb) {
    var xhr = new XMLHttpRequest;
    xhr.open('head', url);
    xhr.onload = function () {
        cb(xhr.getResponseHeader('content-length'));
    };
    xhr.send();
};

MediaData.prototype.fetchNextRange = function (currentSegment, noinit, find) {
    if (!currentSegment) {
        currentSegment = this.currentSegment;
    }
    
    if (!find && (!this.segments[currentSegment] || this.segments[currentSegment][0])) {
        return;
    }
    
    var isRange = false;
    
    if (
        MediaData.userBrowser == "chrome" || 
        MediaData.userBrowser == "safari" || 
        MediaData.userBrowser == "opera"
    ) {
        isRange = true;
    }
    
    // Значительно стабильнее работает без грубой резки на части
    isRange = false;
    
    var countSegments = 0;
    
    var start = 0;
    var end = 0;
    
    var iSegment = parseInt(currentSegment);
    
    if (isRange) {
        isRange = false;
        
        for (var i in this.segments[currentSegment][1]) {
            end = parseFloat(this.sources[this.sKey]['manifest']['media'][this.segments[currentSegment][1][i]][1]);
        }
    
        do {
            iSegment++;
        
            start = end;
        
            if (!this.segments[iSegment]) {
                break;
            }
        
            end = 0;
        
            for (var i in this.segments[iSegment][1]) {
                end = parseFloat(this.sources[this.sKey]['manifest']['media'][this.segments[iSegment][1][i]][1]);
            }
            
            if (iSegment > 3) {
                break;
            }
        
            if (end - start > 3) {
                isRange = true;
            }
        } while (true);
    }
    
    //console.log("isRange: ", isRange);
    
    var time = Date.now();
    
    if (!find && this.lastTimeFetchNextRange && (time - this.lastTimeFetchNextRange > 3000)) {
        this.sKeyMinus();
    }
    
    if (!find) {
        this.lastTimeFetchNextRange = time;
    
        this.segments[currentSegment][0] = true;
    
        this.chunks[currentSegment] = null;
    }
    
    var segments = [];
    
    countSegments = 0;
    
    start = parseInt(this.sources[this.sKey]['manifest']['media'][this.segments[currentSegment][1][0]][0]);
    end = start;
    
    var timeInterval = [
        this.segments[currentSegment][2],
        this.segments[currentSegment][2]
    ];
    
    this.countFetch++;
    
    iSegment = parseInt(currentSegment);
    do {
        if (!this.segments[iSegment]) {
            break;
        }
        
        timeInterval[1] = this.segments[iSegment][2];
        
        if (this.segments[iSegment+1]) {
            if (parseFloat(this.segments[iSegment+1][2]) - parseFloat(this.segments[currentSegment][2]) > MediaData.loadTimeInterval) {
                break;
            }
            
            if (countSegments > 0 && !isRange && 
                parseFloat(this.segments[currentSegment][2]) - parseFloat(this.timeWaiting) < MediaData.loadTimeWaiting) {
                if (timeInterval[1] - timeInterval[0] >= this.countFetch) {
                    break;
                }
            }
            
            if (this.isSeekSet && countSegments > 0 && !isRange && 
                parseFloat(this.segments[currentSegment][2]) - parseFloat(this.currentTime) < MediaData.loadTimeWaiting) {
                if (timeInterval[1] - timeInterval[0] >= this.countFetch) {
                    break;
                }
            }
        }
        
        for (var i in this.segments[iSegment][1]) {
            if (this.sources[this.sKey]['manifest']['media'][parseInt(this.segments[iSegment][1][i])+1]) {
                end = parseInt(this.sources[this.sKey]['manifest']['media'][parseInt(this.segments[iSegment][1][i])+1][0]) - 1;
            } else {
                end = parseFloat(this.sources[this.sKey]['manifest']['size']);
            }
        }
        
        if (this.setStartCurrentTime) {
            break;
        }
        
        countSegments++;
        
        iSegment++;
    } while (true);
    
    //console.log("countSegments: "+countSegments);
    
    var nowsegment;
    
    var keysegment = 0;
    
    if (this.sKeyInit != this.sKey && !noinit) {
        if (!find) {
            this.sKeyInit = this.sKey;
        }
        
        if (this.sources[this.sKey]['manifest']['init'][1] == start) {
            nowsegment = {
                segment: currentSegment,
                mimeCodec: this.getMimeCodec(),
                url: this.sources[this.sKey]['url'], 
                start: parseInt(this.sources[this.sKey]['manifest']['init'][0]), 
                end: end,
                countSegments: countSegments,
                timeInterval: timeInterval,
            };
        } else {
            keysegment = segments.push({
                key: keysegment,                                
                segment: currentSegment,
                mimeCodec: this.getMimeCodec(),
                url: this.sources[this.sKey]['url'], 
                start: parseInt(this.sources[this.sKey]['manifest']['init'][0]), 
                end: parseInt(this.sources[this.sKey]['manifest']['init'][1]) - 1,
                countSegments: countSegments,
                timeInterval: timeInterval,
            });
            
            nowsegment = {
                segment: currentSegment,
                mimeCodec: this.getMimeCodec(),
                url: this.sources[this.sKey]['url'], 
                start: start, 
                end: end,
                countSegments: countSegments,
                timeInterval: timeInterval,
            };
        }
    } else {
        nowsegment = {
            segment: currentSegment,
            mimeCodec: this.getMimeCodec(),
            url: this.sources[this.sKey]['url'], 
            start: start, 
            end: end,
            countSegments: countSegments,
            timeInterval: timeInterval,
        };
    }
    
    if (parseFloat(this.segments[currentSegment][2]) - parseFloat(this.timeWaiting) < MediaData.loadTimeWaiting && !this.setStartCurrentTime && isRange) {
        start = parseInt(nowsegment.start);
        
        var inext = parseInt(parseInt(nowsegment.end - nowsegment.start) / parseFloat(nowsegment.timeInterval[1] - nowsegment.timeInterval[0]));
        
        //inext = 100000; // load for 100 kb
        
        do {
            if ((start + inext < nowsegment.end && nowsegment.end - (start + inext + inext) > 50) && keysegment <= MediaData.partRange) {
                keysegment = segments.push({
                    key: keysegment,
                    segment: nowsegment.segment,
                    mimeCodec: nowsegment.mimeCodec,
                    url: nowsegment.url, 
                    start: start, 
                    end: start + inext - 1,
                    countSegments: nowsegment.countSegments,
                    timeInterval: nowsegment.timeInterval,
                    segmentRange: true,
                });
            } else {
                keysegment = segments.push({
                    key: keysegment,
                    segment: nowsegment.segment,
                    mimeCodec: nowsegment.mimeCodec,
                    url: nowsegment.url,
                    start: start,
                    end: nowsegment.end,
                    countSegments: nowsegment.countSegments,
                    timeInterval: nowsegment.timeInterval,
                    segmentRange: true,
                    segmentRangeLast: true,
                });
                
                break;
            }
            
            start = start + inext;
        } while(true);
    } else {
        nowsegment.key = keysegment;
        keysegment = segments.push(nowsegment);
    }
    
    if (!find) {
        this.setStartCurrentTime = false;
        
        this.addLoad(segments);
    
        this.load();
    }
    
    //console.log("keysegment: "+keysegment);
    
    return segments;
};

MediaData.prototype.addLoad = function(segments) {
    this.iload++;
        
    var isRange = false;
    
    for (var i in segments) {
        segments[i]['keyload'] = this.iload;
        
        if (segments[i].segmentRange) {
            isRange = true;
        }
    }
    
    //console.log("addLoad: ", this.iload);
    
    this.loads[this.iload] = {
        key: this.iload,
        status: "created",
        count: 0,
        segments: segments,
        range: isRange,
    };
};

MediaData.prototype.load = function() {
    var iload = null;
    
    while(iload = this.getLoadKey()){
        this.loadByKey(iload);
    }
};

MediaData.prototype.getLoadKey = function() {
    for (var i in this.loads) {
        if (this.loads[i].segments.length == 0) {
            delete this.loads[i];
            continue;
        }
        
        if (this.loads[i].status == "pending") {
            break;
        }
        
        if (this.loads[i].status == "success") {
            delete this.loads[i];
        }
        
        if (this.loads[i].count > 5) {
            //this.reinit();
            
            //break;
        }
        
        this.loads[i].count++;
        
        return i;
    }
    
    return null;
};

MediaData.prototype.loadByKey = function(key) {
    var $this = this;
    
    this.loads[key].status = "pending";
    
    this.fetchRange(
        this.cloneSegments(this.loads[key].segments),
        function (iSegment, mimeCodec, chunk) {
            $this.addChuck(iSegment, mimeCodec, chunk);
        }, function (status) {
            if ($this.loads[key]) {
                $this.loads[key].status = status;
            
                if ($this.loads[key].status == "success") {
                    delete $this.loads[key];
                }
            }
        }
    );
};

MediaData.prototype.cloneSegments = function(segments) {
    var newsegments = [];
    
    for (var i in segments) {
        newsegments[i] = {};
        
        for (var j in segments[i]) {
            newsegments[i][j] = segments[i][j];
        }
    }
    
    return newsegments;
};

MediaData.prototype.fetchRange = function(segments, cb_response, cb_status, cache) {
    var $this = this;
    
    var segment = null;
    
    if (segments.length > 0) {
        segment = segments.shift();
    }
    
    if (segment) {
        var keyInterval = segment['start']+"-"+segment['end'];
        
        if ($this.dataRanges[keyInterval] && $this.dataRanges[keyInterval].status == "success") {
            //console.log("Cache Range: ", keyInterval);
            
            var response = $this.dataRanges[keyInterval].data;
            
            if (segments.length > 0) {
                if (segment.segmentRange) {
                    $this.fetchResponse(segment, response, cb_response, cb_status, cache);
                        
                    $this.fetchRange(segments, function (iSegment, mimeCodec, chunk, cache) {
                        cb_response(iSegment, mimeCodec, chunk, cache);
                    }, cb_status);
                } else {
                    $this.fetchRange(segments, function (iSegment, mimeCodec, chunk) {
                        cb_response(iSegment, mimeCodec, $this.concatSegments([new Uint8Array(response), new Uint8Array(chunk)]), cache);
                    }, cb_status);
                }
            } else {
                $this.fetchResponse(segment, response, cb_response, cb_status, cache);
            }
            
            this.sKeyInit = this.sKey;
            
            return;
        } 
        
        if (cache) {
            return;
        }
        
        if (MediaData.ajaxType == 'fetch' && 'fetch' in window) {
            this.xhrfetchskey++;
        
            var key = this.xhrfetchskey;
            
            this.xhrfetchs[key] = {
                ctr: new AbortController(),
                range: segments.range ? true : false
            };
            
            fetch(segment.url+(segment.url.indexOf("?")>0?"&":"?")+'bytes='+segment.start+'-'+segment.end, {
                method: 'GET',
           	    headers: {
           	        'Range': 'bytes=' + segment.start + '-' + segment.end,
                },
                signal: this.xhrfetchs[key].ctr.signal,
            }).then((response) => {
                if (response.status >= 200 && response.status < 300) {
                    return response.arrayBuffer();
                } else {
                    let error = new Error(response.statusText);
                    error.response = response;
                    throw error;
                }
            }).then((response) => {
                if (segments.length > 0) {
                    if (segment.segmentRange) {
                        $this.fetchResponse(segment, response, cb_response, cb_status);
                        
                        $this.fetchRange(segments, function (iSegment, mimeCodec, chunk) {
                            cb_response(iSegment, mimeCodec, chunk);
                        }, cb_status);
                    } else {
                        $this.fetchRange(segments, function (iSegment, mimeCodec, chunk) {
                            cb_response(iSegment, mimeCodec, $this.concatSegments([new Uint8Array(response), new Uint8Array(chunk)]));
                        }, cb_status);
                    }
                } else {
                    $this.fetchResponse(segment, response, cb_response, cb_status);
                }
                
                if ($this.xhrfetchs[key]) {
                    delete $this.xhrfetchs[key];
                }
            }).catch((e) => {
                cb_status("error");
                if ($this.xhrfetchs[key]) {
                    delete $this.xhrfetchs[key];
                }
            });
        } else {
            this.xhrskey++;
        
            var key = this.xhrskey;
        
            var xhrLoad = new XMLHttpRequest;
            
            this.xhrs[key] = {
                xhr: xhrLoad,
                range: segments.range ? true : false
            };
            
            xhrLoad.open('get', segment.url+(segment.url.indexOf("?")>0?"&":"?")+'bytes='+segment.start+'-'+segment.end, true);
            xhrLoad.responseType = 'arraybuffer';
            xhrLoad.setRequestHeader('Range', 'bytes=' + segment.start + '-' + segment.end);
            xhrLoad.addEventListener('load', function () { 
                if (segments.length > 0) {
                    if (segment.segmentRange) {
                        $this.fetchResponse(segment, xhrLoad.response, cb_response, cb_status);
                        
                        $this.fetchRange(segments, function (iSegment, mimeCodec, chunk) {
                            cb_response(iSegment, mimeCodec, chunk);
                        }, cb_status);
                    } else {
                        $this.fetchRange(segments, function (iSegment, mimeCodec, chunk) {
                            cb_response(iSegment, mimeCodec, $this.concatSegments([new Uint8Array(xhrLoad.response), new Uint8Array(chunk)]));
                        }, cb_status);
                    }
                } else {
                    $this.fetchResponse(segment, xhrLoad.response, cb_response, cb_status);
                }
            
                if ($this.xhrs[key]) {
                    delete $this.xhrs[key];
                }
            });
            xhrLoad.addEventListener('error', function () {
                cb_status("error");
                if ($this.xhrs[key]) {
                    delete $this.xhrs[key];
                }
            });
            xhrLoad.addEventListener('abort', function () {
                cb_status("abort");
                if ($this.xhrs[key]) {
                    delete $this.xhrs[key];
                }
            });
            
            xhrLoad.send();
        }
    } else {
        cb_status("success");
    }
};

MediaData.prototype.fetchResponse = function(segment, response, cb_response, cb_status, cache) {
    cb_response(this.currentSegment, segment.mimeCodec, response);
    
    if (segment.segmentRange) {
        if (segment.keyload >= 0) {
            if (this.loads[segment.keyload]) {
                if (segment.key >= 0) {
                    if (this.loads[segment.keyload].segments[segment.key]) {
                        delete this.loads[segment.keyload].segments[segment.key];
                    }
                }
                
                if (this.loads[segment.keyload].segments.length == 0) {
                    cb_status("success");
                }
            }
        }
    } 
    
    if (!segment.segmentRange || segment.segmentRangeLast) {
        cb_status("success");
        
        for (var i = parseInt(this.currentSegment); i < parseInt(this.currentSegment) + parseInt(segment.countSegments); i++) {
            this.segments[i][0] = true;
        }
        
        this.currentSegment = parseInt(this.currentSegment) + parseInt(segment.countSegments);
    }
    
    if (this.currentSegment > 0) {
        if (parseFloat(this.segments[this.currentSegment][2]) - parseFloat(this.timeWaiting) < MediaData.loadTimeWaiting) {
            this.timeLoadNext = 0;
            this.isLoadTimeWaiting = true;
        } else if (this.isLoadTimeWaiting) {
            this.timeLoadNext = 0;
            this.isLoadTimeWaiting = false;
        } else {
            this.timeLoadNext = parseFloat(parseFloat(this.segments[parseInt(this.currentSegment)-1][2]) - 
                (parseFloat(segment.timeInterval[1]) - parseFloat(segment.timeInterval[0])) * 0.9);
        }
    } else {
        this.timeLoadNext = 0;
    }
    
    //console.log("timeLoadNext: "+this.timeLoadNext+"/"+this.media.currentTime+"/"+segment.timeInterval[0]+"/"+segment.timeInterval[1]);   
    
    if (!cache) {
        if (this.getLoadKey()) {
            this.load();
        } else if (this.timeLoadNext == 0 || this.media.currentTime >= this.timeLoadNext) {
            this.checkBuffer();
        }
    }
};

MediaData.prototype.concatSegments = function(arrays) {
    let result = [];
    for (let i = 0; i < arrays.length; i++) {
        for (let j = 0; j < arrays[i].length; j++) {
            result.push(arrays[i][j]);
        }
    }
    return new Uint8Array(result);
}

MediaData.prototype.appendSegment = function(mimeCodec, chunk) {
    var $this = this;
    
    if (this.lMimeCodec != mimeCodec) {
        this.lMimeCodec = mimeCodec;
        
        if (this.sourceBuffer) {
            this.sourceBuffer.changeType(mimeCodec);
        }     
    }
    
    try {
        this.sourceBuffer.appendBuffer(chunk);
        
        this.completeupdating(function () {
            setTimeout(function () {
                $this.playControl(null, 500);
            }, 1000);
        });
    } catch (error) {
        console.error(error);
    }
    
    if (!this.canplay) {
        if (this.sourceBuffer.updating) {
            this.completeupdating(function () {
                $this.canplay = true;
            });
        } else {
            this.canplay = true;
        }
    }
};

MediaData.prototype.completeupdating = function(cb) {
    var $this = this;
    
    if (this.sourceBuffer && this.sourceBuffer.updating) {
        setTimeout(function () {
           $this.completeupdating(cb); 
        }, 10);
        
        return;
    }
    
    cb();
};

MediaData.prototype.addChuck = function(iSegment, mimeCodec, chunk) {
    this.chunks[iSegment] = {
        mimeCodec: mimeCodec,
        chunk: chunk,
    };
    
    this.appendChucks();
};

MediaData.prototype.appendChucks = function(chunk) {
    var $this = this;
    
    for (var i in this.chunks) {
        if (!this.chunks[i]) {
            return;
        }
        
        if (!this.sourceBuffer || this.sourceBuffer.updating) {
            setTimeout(function () {
                $this.appendChucks();
            }, 10);
            return;
        }
        
        this.appendSegment(this.chunks[i].mimeCodec, this.chunks[i].chunk);
        delete this.chunks[i];
    }
};

MediaData.prototype.checkBuffer = function(_) {
    var $this = this;
    
    if (this.currentSegment >= this.sources[this.sKey]['manifest']['media'].length && this.haveAllSegments()) {
        //console.log('last segment', this.mediaSource.readyState);
        this.mediaSource.endOfStream();
    } else if (this.shouldFetchNextSegment()) {
        //console.log('time to fetch next chunk', this.media.currentTime);
        
        $this.fetchNextRange();
    }
};

MediaData.prototype.seeking = function(e) {
    var $this = this;
    
    this.timeSeeking = Date.now();
    this.seek = true;
    
    this.countFetch = 0;
        
    //console.log("seeking");
    if (this.mediaSource.readyState === 'open') {
        this.timeWaiting = this.media.currentTime;
        
        for (var i in this.xhrs) {
            if (this.xhrs[i].range) {
                continue;
            }
            
            if (this.xhrs[i].xhr) {
                this.xhrs[i].xhr.abort();
            }
            
            delete this.xhrs[i];
        }
        
        for (var i in this.xhrfetchs) {
            if (this.xhrfetchs[i].range) {
                continue;
            }
            
            if (this.xhrfetchs[i]) {
                this.xhrfetchs[i].ctr.abort();
            }
            
            delete this.xhrfetchs[i];
        }
        
        for (var i in this.loads) {
            if (!this.loads[i].range) {
                delete this.loads[i];
            }
        }
        
        this.chunks = {};
        
        if (this.sourceBuffer) {
            if (!this.sourceBuffer.updating) {
                this.sourceBuffer.remove(0, this.media.duration);
            }
            
            if (MediaData.userBrowser == "firefox") {
                
            } else {
                this.sourceBuffer.abort();
            }
        }
        
        
        for (var i in this.segments) {
            this.segments[i][0] = false;
        }
        
        var currentTime = this.currentTime;
        
        for (var i in this.segments) {
            if (parseFloat(this.segments[i][2]) > parseFloat(this.media.currentTime)) {
                break;
            }
            currentTime = parseFloat(this.segments[i][2]);
            this.currentSegment = i;
        }
        
        this.timeLoadNext = 0;
        
        //console.log("seeking: ", currentTime, "/", this.currentTime, "/", this.media.currentTime);
        
        if (MediaData.userBrowser == "firefox") {
            
        } else {
            this.sourceBuffer.timestampOffset = currentTime;
        }
    } else {
        console.log('seek but not open?');
        console.log(this.mediaSource.readyState);
    }
    
    if (this.isSeekSet) {
        this.isSeekSet = false;
        
        $this.seeked();
    } else {
        this.completeSeeking(function () {
            if ($this.seek) {
                $this.seeked();
            }
    });
    }
};

MediaData.prototype.completeSeeking = function(cb, recurse) {
    var $this = this;
    
    if (this.startCompleteSeeking && !recurse) {
        return;
    }
    
    this.startCompleteSeeking = true;
    
    if (Date.now() - this.timeSeeking < MediaData.loadSeekTime) {
        setTimeout(function () {
           $this.completeSeeking(cb, true); 
        }, 50);
        
        return;
    }
    
    cb();
    
    this.startCompleteSeeking = false;
};

MediaData.prototype.seeked = function(e) {
    var $this = this;
    
    this.seek = false;
    
    //console.log("seeked");
    
    if (this.mediaSource.readyState === 'open') {
        this.checkBuffer(e);
                
        this.play();
        
        //console.log("MediaSource readyState: ", this.mediaSource.readyState);
    } else {
        console.log('seek but not open?');
        console.log(this.mediaSource.readyState);
    }
};

MediaData.prototype.haveAllSegments = function() {
    for (var i in this.segments) {
        if (!this.segments[i][0]) {
            return false;
        }
    }
    
    return true;
};

MediaData.prototype.shouldFetchNextSegment = function() {
    if (this.currentSegment >= 0 && this.currentSegment < this.segments.length) {
        if (this.media.currentTime >= this.timeLoadNext) {
            return true;
        }   
    }
    
    return false;
};

MediaData.randomString = function(lenString) {  
    if (!lenString) {
        lenString = 7;
    }
    
    var characters = "ABCDEFGHIJKLMNOPQRSTUVWXTZabcdefghiklmnopqrstuvwxyz";
    
    var randomstring = '';  
  
    for (var i=0; i < lenString; i++) {  
        var rnum = Math.floor(Math.random() * characters.length);  
        randomstring += characters.substring(rnum, rnum+1);  
    }  
  
    return randomstring;  
};

MediaData.httpBuildQuery = function (obj, num_prefix, temp_key) {

    var output_string = []

    Object.keys(obj).forEach(function (val) {

        var key = val;

        num_prefix && !isNaN(key) ? key = num_prefix + key : ''

        var key = encodeURIComponent(key.replace(/[!'()*]/g, escape));
        temp_key ? key = temp_key + '[' + key + ']' : ''

        if (typeof obj[val] === 'object') {
            var query = MediaData.httpBuildQuery(obj[val], null, key)
            output_string.push(query)
        } else {
            var value = String(obj[val]);

            value = encodeURIComponent(value.replace(/[!'()*]/g, escape));

            output_string.push(key + '=' + value)
        }

    })

    return output_string.join('&')
}

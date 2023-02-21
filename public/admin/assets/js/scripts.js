$(document).ready(function(){ 
    $("#select_items_all").click(function(){
        if($("#select_items_all").prop('checked')){
            $(".select_items").prop('checked', true);
        } else {
            $(".select_items").prop('checked', false);
        }
    });
});

function confirmClick (obj) {
    if (confirm('Подтвердите действие')) {
        location = $(obj).attr('data-href');
    }
};

function changeFindField (obj) {
    event.preventDefault();
    if (event.stopPropagation) {
        event.stopPropagation()
    } else {
        event.cancelBubble = true
    }
    
    changeGetValueByPage($(obj).attr('name'), $(obj).val());
    
    return false;
}

function changeSortField (obj) {
    event.preventDefault();
    if (event.stopPropagation) {
        event.stopPropagation()
    } else {
        event.cancelBubble = true
    }
    
    changeGetValueByPage($(obj).attr('data-name'), $(obj).attr('data-value'));
    
    return false;
}

function changeGetValueByPage(name, value) {
    var data = {};

    var link = location.href;

    var purl = parse_url(link);

    if (purl && purl['query']) {
        var pstr = parse_str(purl['query']);

        if (pstr[name]) {
            delete pstr[name];
        }

        for (var i in pstr) {
            if (pstr[i] != '')
                data[i] = pstr[i];
        }
    }

    if (value != '')
        data[name] = encodeURIComponent(value.replace(/[!'()*]/g, escape));

    var query = http_build_query(data);
    
    link = link.replace(/\?.*$/, '');
    link = link.replace(/#.*$/, '');

    if (query) {
        link += '?' + query;
    }

    if (location.href != link) {
        //alert(link);

        location.href = link;
    }
}

function parse_str(str, array){	// Parses the string into variables
	// 
	// +   original by: Cagri Ekin
	// +   improved by: Michael White (http://crestidg.com)

	var glue1 = '=';
	var glue2 = '&';

	var array2 = str.split(glue2);
	var array3 = [];
	for(var x=0; x<array2.length; x++){
		var tmp = array2[x].split(glue1);
		array3[unescape(tmp[0])] = unescape(tmp[1]).replace(/[+]/g, ' ');
	}

	if(array){
		array = array3;
	} else{
		return array3;
	}
}



var parse_url = function (str, component) {
    // example 1: parse_url('http://username:password@hostname/path?arg=value#anchor');
    // returns 1: {scheme: 'http', host: 'hostname', user: 'username', pass: 'password', path: '/path', query: 'arg=value', fragment: 'anchor'}

    var query, key = [
            'source', 'scheme', 'authority', 'userInfo',
            'user', 'pass', 'host', 'port', 'relative', 'path',
            'directory', 'file', 'query', 'fragment'
        ],
        ini = (this.php_js && this.php_js.ini) || {},
        mode = (ini['phpjs.parse_url.mode'] &&
            ini['phpjs.parse_url.mode'].local_value) || 'php',
        parser = {
            php: /^(?:([^:\/?#]+):)?(?:\/\/()(?:(?:()(?:([^:@]*):?([^:@]*))?@)?([^:\/?#]*)(?::(\d*))?))?()(?:(()(?:(?:[^?#\/]*\/)*)()(?:[^?#]*))(?:\?([^#]*))?(?:#(.*))?)/,
            strict: /^(?:([^:\/?#]+):)?(?:\/\/((?:(([^:@]*):?([^:@]*))?@)?([^:\/?#]*)(?::(\d*))?))?((((?:[^?#\/]*\/)*)([^?#]*))(?:\?([^#]*))?(?:#(.*))?)/,
            loose: /^(?:(?![^:@]+:[^:@\/]*@)([^:\/?#.]+):)?(?:\/\/\/?)?((?:(([^:@]*):?([^:@]*))?@)?([^:\/?#]*)(?::(\d*))?)(((\/(?:[^?#](?![^?#\/]*\.[^?#\/.]+(?:[?#]|$)))*\/?)?([^?#\/]*))(?:\?([^#]*))?(?:#(.*))?)/ // Added one optional slash to post-scheme to catch file:/// (should restrict this)
        };

    var m = parser[mode].exec(str),
        uri = {},
        i = 14;
    while (i--) {
        if (m[i]) {
            uri[key[i]] = m[i];
        }
    }

    if (component) {
        return uri[component.replace('PHP_URL_', '').toLowerCase()];
    }
    if (mode !== 'php') {
        var name = (ini['phpjs.parse_url.queryKey'] &&
            ini['phpjs.parse_url.queryKey'].local_value) || 'queryKey';
        parser = /(?:^|&)([^&=]*)=?([^&]*)/g;
        uri[name] = {};
        query = uri[key[12]] || '';
        query.replace(parser, function ($0, $1, $2) {
            if ($1) {
                uri[name][$1] = $2;
            }
        });
    }
    delete uri.source;
    return uri;
}

var http_build_query = function (obj, num_prefix, temp_key) {

    var output_string = []

    Object.keys(obj).forEach(function (val) {

        var key = val;

        num_prefix && !isNaN(key) ? key = num_prefix + key : ''

        var key = encodeURIComponent(key.replace(/[!'()*]/g, escape));
        temp_key ? key = temp_key + '[' + key + ']' : ''

        if (typeof obj[val] === 'object') {
            var query = http_build_query(obj[val], null, key)
            output_string.push(query)
        } else {
            var value = String(obj[val]);

            value = encodeURIComponent(value.replace(/[!'()*]/g, escape));

            output_string.push(key + '=' + value)
        }

    })

    return output_string.join('&')

}

var stripos = function (f_haystack, f_needle, f_offset) { // Find position of first occurrence of a case-insensitive string
    //
    // +	 original by: Martijn Wieringa

    var haystack = f_haystack.toLowerCase();
    var needle = f_needle.toLowerCase();
    var index = 0;

    if (f_offset == undefined) {
        f_offset = 0;
    }

    if ((index = haystack.indexOf(needle, f_offset)) > -1) {
        return index;
    }

    return false;
}
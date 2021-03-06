package admin

var varIndex = `
<!DOCTYPE html>
<html lang="en">
<head>
    <!-- Load required Bootstrap and BootstrapVue CSS -->
    <link type="text/css" rel="stylesheet" href="//unpkg.com/bootstrap/dist/css/bootstrap.min.css" />
    <link type="text/css" rel="stylesheet" href="//unpkg.com/bootstrap-vue@latest/dist/bootstrap-vue.min.css" />

    <!-- Load polyfills to support older browsers -->
    <script src="//polyfill.io/v3/polyfill.min.js?features=es2015%2CMutationObserver"></script>

    <!-- Load Vue followed by BootstrapVue -->
    <script src="//unpkg.com/vue@latest/dist/vue.min.js"></script>
    <script src="//unpkg.com/bootstrap-vue@latest/dist/bootstrap-vue.min.js"></script>
    <script src="//unpkg.com/axios/dist/axios.min.js"></script>
    <meta charset="UTF-8">
    <title>Vars</title>
</head>
<body>

<div id="app">
	<b-navbar>
		<b-navbar-brand>Variables</b-navbar-brand>
		<b-navbar-nav>
			<b-nav-item v-for="item in nav" :href="item.url" :active="item.name === 'Variables'">{{ item.name }}</b-nav-item>
		</b-navbar-nav>
	</b-navbar>
    <b-alert
            dismissable
            variant="error"
            v-if="err"
            @dismissed="err = ''">
        {{ err }}
    </b-alert>
    <b-container>
        <b-table
                fixed
                :items="vars"
                :sort-by.sync="sortBy"
                :fields="fields"></b-table>
    </b-container>
</div>

<script>
    var app = new Vue({
        el: '#app',
        data: {
            err: '',
			nav: [],
            vars: [],
            sortBy: 'key',
            fields: [
                { key: { sortable: true } },
                'value'
            ]
        },
        mounted() {
            this.getData();
            axios.get('/nav')
                .then(resp => {
                    this.nav = resp.data;
                })
                .catch(err => console.log(err))
        },
        methods: {
            getData: function() {
                axios.get('/vars/api')
                    .then(resp => {
                        this.vars = resp.data;
                    })
                    .catch(err => this.err = err);
            }
        }
    })
</script>
</body>
</html>
`

var apppassIndex = `
<!DOCTYPE html>
<!DOCTYPE html>
<html lang="en">
<head>
<!-- Load required Bootstrap and BootstrapVue CSS -->
<link type="text/css" rel="stylesheet" href="//unpkg.com/bootstrap/dist/css/bootstrap.min.css"/>
<link type="text/css" rel="stylesheet" href="//unpkg.com/bootstrap-vue@latest/dist/bootstrap-vue.min.css"/>

<!-- Load polyfills to support older browsers -->
<script src="//polyfill.io/v3/polyfill.min.js?features=es2015%2CMutationObserver"></script>

<!-- Load Vue followed by BootstrapVue -->
<script src="//unpkg.com/vue@latest/dist/vue.min.js"></script>
<script src="//unpkg.com/bootstrap-vue@latest/dist/bootstrap-vue.min.js"></script>
<script src="//unpkg.com/axios/dist/axios.min.js"></script>
<meta charset="UTF-8">
<title>Vars</title>
</head>
<body>

<div id="app">
<b-navbar>
<b-navbar-brand>App Pass</b-navbar-brand>
<b-navbar-nav>
<b-nav-item v-for="item in nav" :href="item.url" :active="item.name === 'App Pass'">{{ item.name }}
</b-nav-item>
</b-navbar-nav>
</b-navbar>
<div class="alert alert-warning alert-dismissible fade show" role="alert" v-if="err != ''">
<b-button type="link" class="close" data-dismiss="alert" aria-label="Close" @click="err = ''">
<span aria-hidden="true">&times;</span>
</b-button>
{{ err }}
</div>
<b-form>
<b-container>
<b-row>
<b-col cols="5">Password:</b-col>
<b-col>
<b-input v-model="password"/>
</b-col>
</b-row>
<b-row>
<b-col cols="5">Secret:</b-col>
<b-col>
<b-input v-model="entry.secret"/>
</b-col>
</b-row>
<b-row>
<b-col>
<b-button @click="list">List</b-button>
</b-col>
<b-col>
<b-button @click="newPass">New</b-button>
</b-col>
</b-row>
</b-container>
</b-form>
<b-container v-show="showPassword" style="padding: 2em">
<b-row align-h="center">
<b-col align-self="center" cols="1">ID:</b-col>
<b-col align-self="center" cols="3">{{ entry.id }}</b-col>
</b-row>
<b-row align-h="center">
<b-col align-self="center" cols="1">Password:</b-col>
<b-col align-self="center" cols="3">{{ entry.secret }}:{{ showPassword }}</b-col>
</b-row>
<b-row align-h="center">
<b-col align-self="center" class="text-center" cols="6">Note: this password will only be displayed once. For single-field entry passwords, use the secret:password format.</b-col>
</b-row>
</b-container>
<b-container>
<b-row style="padding-top: 2em;">
<b-col>
<ul>
<li v-for="entry in entries" key="id">
<a @click="rm(entry)" href="#">X</a> {{entry.id}}</li>
</ul>
</b-col>
</b-row>
</b-container>
</div>

<script>
var app = new Vue({
el: '#app',
data: {
err: '',
entry: {
secret: '',
},
password: '',
showPassword: '',
nav: [],
entries: [],
},
mounted() {
axios.get('/nav')
.then(resp => {
this.nav = resp.data;
})
.catch(err => console.log(err))
},
methods: {
rm: function (data) {
this.showPassword = '';
this.entry.id = data.id
axios.delete('/apppass/api', {
data: {
password: this.password,
passEntry: this.entry
}
})
.then(() => {
this.getData()
})
.catch(({response}) => {
console.log('error: ' + response.data.err)
this.err = response.data.err
})
},
list: function () {
this.showPassword = '';
this.getData();
},
newPass: function () {
axios.put('/apppass/api', {
password: this.password,
passEntry: this.entry
})
.then(resp => {
this.getData()
this.showPassword = resp.data.pass
this.entry.id = resp.data.id
})
.catch(({response}) => {
console.log('error: ' + response.data.err)
this.err = response.data.err
})
},
getData: function () {
axios.post('/apppass/api', {
password: this.password,
passEntry: this.entry
})
.then(resp => {
this.entries = resp.data;
})
.catch(({response}) => {
console.log('error: ' + response.data.err)
this.err = response.data.err
})
}
}
})
</script>
</body>
</html>
`

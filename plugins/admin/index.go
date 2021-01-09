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

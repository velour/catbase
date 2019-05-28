// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package fact

// I hate this, but I'm creating strings of the templates to avoid having to
// track where templates reside.

// 2016-01-15 Later note, why are these in plugins and the server is in bot?

var factoidIndex = `
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
    <script src="https://unpkg.com/vue-router"></script>
    <script src="https://unpkg.com/axios/dist/axios.min.js"></script>
    <meta charset="UTF-8">
    <title>Factoids</title>
</head>
<body>

<div id="app">
    <h1>Factoids</h1>
    <b-alert
            dismissable
            variant="error"
            v-if="err"
            @dismissed="err = ''">
        {{ err }}
    </b-alert>
    <b-form @submit="runQuery">
    <b-container>
        <b-row>
            <b-col cols="10">
                <b-input v-model="query"></b-input>
            </b-col>
            <b-col cols="2">
                <b-button>Search</b-button>
            </b-col>
        </b-row>
        <b-row>
            <b-col>
            <b-table
                    fixed
                    :items="results"
                    :fields="fields"></b-table>
            </b-col>
        </b-row>
    </b-container>
    </b-form>
</div>

<script>
    var router = new VueRouter({
    mode: 'history',
    routes: []
    });
    var app = new Vue({
        el: '#app',
        router,
        data: {
            err: '',
            query: '',
            results: [],
            fields: [
                { key: 'Fact', sortable: true },
                { key: 'Tidbit', sortable: true },
                { key: 'Owner', sortable: true },
                { key: 'Count' }
            ]
        },
        mounted() {
            if (this.$route.query.query) {
                this.query = this.$route.query.query;
                this.runQuery()
            }
        },
        computed: {
            result0: function() {
                return this.results[0] || "";
            }
        },
        methods: {
            runQuery: function(evt) {
                if (evt) {
                    evt.preventDefault();
                    evt.stopPropagation()
                }
                axios.post('/factoid/api', {query: this.query})
                    .then(resp => {
                        this.results = resp.data;
                    })
                    .catch(err => (this.err = err));
            }
        }
    })
</script>
</body>
</html>
`

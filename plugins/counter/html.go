package counter

var html = `
<html>
    <head>
        <!-- Load required Bootstrap and BootstrapVue CSS -->
        <link type="text/css" rel="stylesheet" href="//unpkg.com/bootstrap/dist/css/bootstrap.min.css" />
        <link type="text/css" rel="stylesheet" href="//unpkg.com/bootstrap-vue@latest/dist/bootstrap-vue.min.css" />

        <!-- Load polyfills to support older browsers -->
        <script src="//polyfill.io/v3/polyfill.min.js?features=es2015%2CMutationObserver"></script>

        <!-- Load Vue followed by BootstrapVue -->
        <script src="//unpkg.com/vue@latest/dist/vue.min.js"></script>
        <script src="//unpkg.com/bootstrap-vue@latest/dist/bootstrap-vue.min.js"></script>
        <script src="https://unpkg.com/axios/dist/axios.min.js"></script>
		<title>Counters</title>
    </head>
    <body>

        <div id="app">
			<b-navbar>
				<b-navbar-brand>Counters</b-navbar-brand>
				<b-navbar-nav>
					<b-nav-item v-for="item in nav" :href="item.url" :active="item.name === 'Counter'">{{ item.name }}</b-nav-item>
				</b-navbar-nav>
			</b-navbar>
            <b-alert
                dismissable
				:show="err"
                variant="error">
                    {{ err }}
            </b-alert>
            <b-container>
                <b-row>
                    <b-col cols="5">Password:</b-col>
                    <b-col><b-input v-model="answer"></b-col>
                </b-row>
                <b-row v-for="(counter, user) in counters">
                    {{ user }}:
                    <b-container>
                        <b-row v-for="(count, thing) in counter">
                            <b-col offset="1">
                            {{ thing }}:
                            </b-col>
                            <b-col>
                                {{ count }}
                            </b-col>
                            <b-col cols="2">
                                <button @click="subtract(user,thing,count)">-</button>
                                <button @click="add(user,thing,count)">+</button>
                            </b-col>
                        </b-row>
                    </b-container>
                </b-row>
            </b-container>
        </div>

        <script>
		function convertData(data) {
			var newData = {};
			for (let i = 0; i < data.length; i++) {
				let entry = data[i]
				if (newData[entry.Nick] === undefined) {
					newData[entry.Nick] = {}
				}
				newData[entry.Nick][entry.Item] = entry.Count;
			}
			return newData;
		}
        var app = new Vue({
        	el: '#app',
        	data: {
                err: '',
				nav: [],
                answer: '',
                correct: 0,
                counters: {}
        	},
            mounted() {
				axios.get('/nav')
					.then(resp => {
						this.nav = resp.data;
					})
                .catch(err => console.log(err))
                axios.get('/counter/api')
                    .then(resp => (this.counters = convertData(resp.data)))
                    .catch(err => (this.err = err));
            },
        	methods: {
        		add(user, thing, count) {
					axios.post('/counter/api',
						{user: user, thing: thing, action: '++', password: this.answer})
						.then(resp => {this.counters = convertData(resp.data); this.err = '';})
						.catch(err => this.err = err);
                },
        		subtract(user, thing, count) {
					axios.post('/counter/api',
						{user: user, thing: thing, action: '--', password: this.answer})
						.then(resp => {this.counters = convertData(resp.data); this.err = '';})
						.catch(err => this.err = err);
                }
        	}
        })
        </script>
    </body>
</html>
`

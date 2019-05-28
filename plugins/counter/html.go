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
			<h1>Counters</h1>
            <b-alert
                dismissable
                variant="error"
                v-if="err"
                @dismissed="err = ''">
                    {{ err }}
            </b-alert>
            <b-container>
                <b-row>
                    <b-col cols="5">Human test: What is {{ equation }}?</b-col>
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
                                <button :disabled="!authenticated" @click="subtract(user,thing,count)">-</button>
                                <button :disabled="!authenticated" @click="add(user,thing,count)">+</button>
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
                answer: '',
                correct: 0,
                err: '',
                counters: {
                    stk5: {
                        beer: 12,
                        tea: 84,
                        coffee: 127
                    },
                    flyngpngn: {
                        beer: 123,
                        mead: 1,
                        tea: 130
                    }
                }
        	},
            mounted() {
                axios.get('/counter/api')
                    .then(resp => (this.counters = convertData(resp.data)))
                    .catch(err => (this.err = err));
            },
            computed: {
                authenticated: function() {
                    if (Number(this.answer) === this.correct)
                        return true;
                    return false;
                },
                equation: function() {
                    const x = Math.floor(Math.random() * 100);
                    const y = Math.floor(Math.random() * 100);
                    const z = Math.floor(Math.random() * 100);
                    const ops = ['+', '-', '*'];
                    const op1 = ops[Math.floor(Math.random()*3)];
                    const op2 = ops[Math.floor(Math.random()*3)];
                    const eq = ""+x+op1+y+op2+z;
                    this.correct = eval(eq);
                    return eq
                }
            },
        	methods: {
        		add(user, thing, count) {
                    this.counters[user][thing]++;
					axios.post('/counter/api',
						{user: user, thing: thing, action: '++'})
						.then(resp => (this.counters = convertData(resp.data)))
						.catch(err => (this.err = err));
                },
        		subtract(user, thing, count) {
                    this.counters[user][thing]--;
					axios.post('/counter/api',
						{user: user, thing: thing, action: '--'})
						.then(resp => (this.counters = convertData(resp.data)))
						.catch(err => (this.err = err));
                }
        	}
        })
        </script>
    </body>
</html>
`

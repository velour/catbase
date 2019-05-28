package cli

var indexHTML = `
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
    <script src="https://unpkg.com/axios/dist/axios.min.js"></script>
    <meta charset="UTF-8">
    <title>CLI</title>
</head>
<body>

<div id="app">
    <h1>CLI</h1>
    <b-alert
            dismissable
            variant="error"
            v-if="err"
            @dismissed="err = ''">
        {{ err }}
    </b-alert>
    <b-container>
        <b-row>
            <b-form-group
                    :label="humanTest"
                    label-for="input-1"
                    label-cols="8"
                    autofocus>
                <b-input v-model="answer" id="input-1" autocomplete="off"></b-input>
            </b-form-group>
        </b-row>
        <b-row>
            <b-form-textarea
                    v-sticky-scroll
                    disabled
                    id="textarea"
                    v-model="text"
                    placeholder="The bot will respond here..."
                    rows="10"
                    max-rows="10"
                    no-resize
            ></b-form-textarea>
        </b-row>
            <b-form
                @submit="send">
                <b-row>
                <b-col>
                    <b-form-input
                            type="text"
                            placeholder="Username"
                            v-model="user"></b-form-input>
                </b-col>
                <b-col>
                    <b-form-input
                            type="text"
                            placeholder="Enter something to send to the bot"
                            v-model="input"
                            autocomplete="off"
                    ></b-form-input>
                </b-col>
                <b-col>
                    <b-button type="submit" :disabled="!authenticated">Send</b-button>
                </b-col>
                </b-row>
            </b-form>
    </b-container>
</div>

<script>
    var app = new Vue({
        el: '#app',
        data: {
            answer: '',
            correct: 0,
            textarea: [],
            user: '',
            input: '',
            err: '',
        },
        computed: {
            authenticated: function() {
                if (Number(this.answer) === this.correct && this.user !== '')
                    return true;
                return false;
            },
            humanTest: function() {
                const x = Math.floor(Math.random() * 100);
                const y = Math.floor(Math.random() * 100);
                const z = Math.floor(Math.random() * 100);
                const ops = ['+', '-', '*'];
                const op1 = ops[Math.floor(Math.random()*3)];
                const op2 = ops[Math.floor(Math.random()*3)];
                const eq = ""+x+op1+y+op2+z;
                this.correct = eval(eq);
                return "Human test: What is " + eq + "?";
            },
            text: function() {
                return this.textarea.join('\n');
            }
        },
        methods: {
            addText(user, text) {
                this.textarea.push(user + ": " + text);
                const len = this.textarea.length;
                if (this.textarea.length > 10)
                    this.textarea = this.textarea.slice(len-10, len);
            },
            send(evt) {
                evt.preventDefault();
				evt.stopPropagation()
				this.input = "";
                if (!this.authenticated) {
                    console.log("User is a bot.");
                    this.err = "User appears to be a bot.";
                    return;
                }
                const payload = {user: this.user, payload: this.input};
                console.log("Would have posted to /cli/api:" + JSON.stringify(payload));
                this.addText(this.user, this.input);
                axios.post('/cli/api', payload)
                    .then(resp => {
                        console.log(JSON.stringify(resp.data));
                        const data = resp.data;
                        this.addText(data.user, data.payload.trim());
                    })
                    .catch(err => (this.err = err));
            }
        }
    })
</script>
</body>
</html>
`

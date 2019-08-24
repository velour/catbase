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
	<b-navbar>
		<b-navbar-brand>CLI</b-navbar-brand>
		<b-navbar-nav>
			<b-nav-item v-for="item in nav" :href="item.URL" :active="item.Name === 'CLI'">{{ "{{ item.Name }}" }}</b-nav-item>
		</b-navbar-nav>
	</b-navbar>
    <b-alert
            dismissable
            variant="error"
			:show="err">
        {{ "{{ err }}" }}
    </b-alert>
    <b-container>
		<b-row>
			<b-col cols="5">Password:</b-col>
			<b-col><b-input v-model="answer"></b-col>
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
            err: '',
			nav: {{ .Nav }},
            answer: '',
            correct: 0,
            textarea: [],
            user: '',
            input: '',
        },
        computed: {
            authenticated: function() {
                if (this.user !== '')
                    return true;
                return false;
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
                if (!this.authenticated) {
                    return;
                }
                const payload = {user: this.user, payload: this.input, password: this.answer};
                this.addText(this.user, this.input);
				this.input = "";
                axios.post('/cli/api', payload)
                    .then(resp => {
                        const data = resp.data;
                        this.addText(data.user, data.payload.trim());
						this.err = '';
                    })
                    .catch(err => (this.err = err));
            }
        }
    })
</script>
</body>
</html>
`

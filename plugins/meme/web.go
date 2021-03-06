package meme

var memeIndex = `
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
    <title>Memes</title>
</head>
<body>

<div id="app">
    <b-navbar>
        <b-navbar-brand>Memes</b-navbar-brand>
        <b-navbar-nav>
            <b-nav-item v-for="item in nav" :href="item.url" :active="item.name === 'Meme'">{{ item.name }}</b-nav-item>
        </b-navbar-nav>
    </b-navbar>
    <b-alert
            dismissable
            variant="error"
            v-if="err"
            @dismissed="err = ''">
        {{ err }}
    </b-alert>
    <b-form @submit="saveConfig" v-if="editConfig">
        <b-container>
            <b-row>
                <b-col cols="1">
                    Name:
                </b-col>
                <b-col>
                    {{ editConfig.name }}
                </b-col>
            </b-row>
            <b-row>
                <b-col cols="1">
                    Image:
                </b-col>
                <b-col>
                    <img :src="editConfig.url" :alt="editConfig.url" rounded block fluid />
                </b-col>
            </b-row>
            <b-row>
                <b-col cols="1">
                    URL:
                </b-col>
                <b-col>
                    <b-input placeholder="URL..." v-model="editConfig.url"></b-input>
                </b-col>
            </b-row>
            <b-row>
                <b-col cols="1">
                    Config:
                </b-col>
                <b-col>
                    <b-form-textarea v-model="editConfig.config" rows="10">
                    </b-form-textarea>
                </b-col>
            </b-row>
            <b-row>
                <b-button type="submit" variant="primary">Save</b-button>
                &nbsp;
                <b-button @click="rm" variant="danger">Delete</b-button>
                &nbsp;
                <b-button type="cancel" @click="editConfig = null" variant="secondary">Cancel</b-button>
            </b-row>
        </b-container>
    </b-form>
    <b-form @submit="addMeme" v-if="!editConfig">
        <b-container>
            <b-row>
                <b-col cols="3">
                    <b-input placeholder="Name..." v-model="name"></b-input>
                </b-col>
                <b-col cols="3">
                    <b-input placeholder="URL..." v-model="url"></b-input>
                </b-col>
                <b-col cols="3">
                    <b-input placeholder="Config..." v-model="config"></b-input>
                </b-col>
                <b-col cols="3">
                    <b-button type="submit">Add Meme</b-button>
                </b-col>
            </b-row>
            <b-row>
                <b-col>
                    <b-table
                            fixed
                            :items="results"
                            :fields="fields">
                        <template v-slot:cell(config)="data">
                            <pre>{{data.item.config | pretty}}</pre>
                            <b-button @click="startEdit(data.item)">Edit</b-button>
                        </template>
                        <template v-slot:cell(image)="data">
                            <b-img :src="data.item.url" rounded block fluid />
                        </template>
                    </b-table>
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
            nav: [],
            name: '',
            url: '',
            config: '',
            results: [],
            editConfig: null,
            fields: [
                { key: 'name', sortable: true },
                { key: 'config' },
                { key: 'image' }
            ]
        },
        mounted() {
            axios.get('/nav')
                .then(resp => {
                    this.nav = resp.data;
                })
                .catch(err => console.log(err))
            this.refresh();
        },
		filters: {
			pretty: function(value) {
				if (!value) {
					return ""
				}
				return JSON.stringify(JSON.parse(value), null, 2);
			}
		},
        methods: {
            refresh: function() {
                axios.get('/meme/all')
                    .catch(err => (this.err = err))
                    .then(resp => {
                        this.results = resp.data
                    })
            },
            addMeme: function(evt) {
                if (evt) {
                    evt.preventDefault();
                    evt.stopPropagation()
                }
                if (this.name && this.url)
                    axios.post('/meme/add', {name: this.name, url: this.url, config: this.config})
                        .then(resp => {
                            this.results = resp.data;
                            this.name = "";
                            this.url = "";
                            this.config = "";
                            this.refresh();
                        })
                        .catch(err => (this.err = err));
            },
            startEdit: function(item) {
                this.editConfig = item;
            },
            saveConfig: function(evt) {
                if (evt) {
                    evt.preventDefault();
                    evt.stopPropagation();
                }
                if (this.editConfig && this.editConfig.name && this.editConfig.url) {
                    axios.post('/meme/add', this.editConfig)
                    .then(resp => {
                        this.results = resp.data;
                        this.editConfig = null;
                        this.refresh();
                    })
                    .catch(err => this.err = err);
                }
            },
			rm: function(evt) {
				if (evt) {
					evt.preventDefault();
					evt.stopPropagation();
				}
				if (confirm("Are you sure you want to delete this meme?")) {
					axios.delete('/meme/rm', { data: this.editConfig })
						.then(resp => {
							this.editConfig = null;
							this.refresh();
						})
						.catch(err => this.err = err);
				}
			}
        }
    })
</script>
</body>
</html>
`

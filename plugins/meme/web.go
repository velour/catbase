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
            <b-nav-item v-for="item in nav" :href="item.URL" :active="item.Name === 'Meme'">{{ "{{ item.Name }}" }}</b-nav-item>
        </b-navbar-nav>
    </b-navbar>
    <b-alert
            dismissable
            variant="error"
            v-if="err"
            @dismissed="err = ''">
        {{ "{{ err }}" }}
    </b-alert>
    <b-form @submit="addMeme">
        <b-container>
            <b-row>
                <b-col cols="5">
                    <b-input placeholder="Name..." v-model="name"></b-input>
                </b-col>
                <b-col cols="5">
                    <b-input placeholder="URL..." v-model="url"></b-input>
                </b-col>
                <b-col cols="2">
                    <b-button type="submit">Add Meme</b-button>
                </b-col>
            </b-row>
            <b-row>
                <b-col>
                    <b-table
                            fixed
                            :items="results"
                            :fields="fields">
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
      nav: {{ .Nav }},
      name: "",
      url: "",
      results: [],
      fields: [
        { key: 'name', sortable: true },
        { key: 'url', sortable: true },
        { key: 'image' }
      ]
    },
    mounted() {
        this.refresh();
    },
    computed: {
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
            axios.post('/meme/add', {Name: this.name, URL: this.url})
              .then(resp => {
                this.results = resp.data;
                this.name = "";
                this.url = "";
                this.refresh();
              })
              .catch(err => (this.err = err));
      }
    }
  })
</script>
</body>
</html>
`

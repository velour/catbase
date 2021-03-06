package secrets

var indexTpl = `
<!DOCTYPE html>
<html lang="en">
<head>
    <!-- Load required Bootstrap and BootstrapVue CSS -->
    <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/bootstrap@5.0.1/dist/css/bootstrap.min.css">
    <link type="text/css" rel="stylesheet" href="//cdn.jsdelivr.net/npm/bootstrap-vue@2.21.2/dist/bootstrap-vue.min.css"/>

    <!-- Load polyfills to support older browsers -->
    <script src="//polyfill.io/v3/polyfill.min.js?features=es2015%2CMutationObserver"></script>

    <!-- Load Vue followed by BootstrapVue -->
    <script src="//cdn.jsdelivr.net/npm/vue"></script>
    <script src="//cdn.jsdelivr.net/npm/bootstrap-vue@2.21.2/dist/bootstrap-vue.js"></script>
    <script src="//cdn.jsdelivr.net/npm/bootstrap-vue@2.21.2/dist/bootstrap-vue-icons.js"></script>
    <script src="//cdn.jsdelivr.net/npm/vue-router@3.5.1/dist/vue-router.min.js"></script>
    <script src="//cdn.jsdelivr.net/npm/axios@0.21.1/dist/axios.min.js"></script>
    <meta charset="UTF-8">
    <title>Memes</title>
</head>
<body>

<div id="app">
    <b-navbar>
        <b-navbar-brand>Memes</b-navbar-brand>
        <b-navbar-nav>
            <b-nav-item v-for="item in nav" :href="item.url" :active="item.name === 'Meme'" :key="item.key">{{ item.name }}</b-nav-item>
        </b-navbar-nav>
    </b-navbar>
    <b-alert
            dismissable
            variant="error"
            :show="err != ''"
            @dismissed="err = ''">
        {{ err }}
    </b-alert>
    <b-form @submit="add">
        <b-container>
            <b-row>
                <b-col cols="3">
                    <b-input placeholder="Key..." v-model="secret.key"></b-input>
                </b-col>
                <b-col cols="3">
                    <b-input placeholder="Value..." v-model="secret.value"></b-input>
                </b-col>
                <b-col cols="3">
                    <b-button type="submit">Add Secret</b-button>
                </b-col>
            </b-row>
            <b-row style="padding-top: 2em;">
                <b-col>
                    <ul>
                        <li v-for="key in results" key="key"><a @click="rm(key)" href="#">X</a> {{key}}</li>
                    </ul>
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
      secret: {key: '', value: ''},
      results: [],
      fields: [
        {key: 'key', sortable: true},
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
    methods: {
      refresh: function () {
        axios.get('/secrets/all')
          .then(resp => {
            this.results = resp.data
            this.err = ''
          })
          .catch(err => (this.err = err))
      },
      add: function (evt) {
        if (evt) {
          evt.preventDefault();
          evt.stopPropagation();
        }
        axios.post('/secrets/add', this.secret)
          .then(resp => {
            this.results = resp.data;
            this.secret.key = '';
            this.secret.value = '';
            this.refresh();
          })
          .catch(err => this.err = err)
      },
      rm: function (key) {
        if (confirm("Are you sure you want to delete this meme?")) {
          axios.delete('/secrets/remove', {data: {key: key}})
            .then(resp => {
              this.refresh();
            })
            .catch(err => this.err = err)
        }
      }
    }
  })
</script>
</body>
</html>
`

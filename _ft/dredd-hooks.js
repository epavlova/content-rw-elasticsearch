let hooks = require('hooks');
let http = require('http');
let fs = require('fs');

const defaultFixtures = './_ft/ersatz-fixtures.yml';

hooks.beforeAll(function (t, done) {
    if (!fs.existsSync(defaultFixtures)) {
        console.log('No fixtures found, skipping hook.');
        done();
        return;
    }

    let contents = fs.readFileSync(defaultFixtures, 'utf8');

    const options = {
        host: 'localhost',
        port: '9000',
        path: '/__configure',
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-yaml'
        }
    };

    let req = http.request(options, function (res) {
        res.setEncoding('utf8');
    });

    req.write(contents);
    req.end();
    hooks.log("Waiting before releasing")
    setTimeout(function () {
        done();
    }, 2000)

});

hooks.beforeEach(function (transaction) {
    // see https://github.com/apiaryio/dredd/blob/master/docs/hooks-nodejs.md#modifying-transaction-request-body-prior-to-execution
    // and because of https://github.com/apiaryio/dredd/blob/master/docs/how-it-works.md#swagger-2
    // "By default Dredd tests only responses with 2xx status codes. Responses with other codes are marked as skipped and can be activated in hooks"
    if (transaction.name.startsWith("Health > /__gtg")) {
        hooks.log("skipping: " + transaction.name);
        transaction.skip = true;
    }
});

///<reference path="../node_modules/grafana-sdk-mocks/app/headers/common.d.ts" />
import _ from 'lodash';
import map from 'lodash/map';
import isObject from 'lodash/isObject';
import filter from 'lodash/filter';
import isUndefined from 'lodash/isUndefined';

export class ConsulDatasource {

    name: string;
    id: string;
    debug: boolean = false;

    /** @ngInject **/
    constructor(instanceSettings, private $q, private backendSrv, private templateSrv) {
        this.name = instanceSettings.name;
        this.id = instanceSettings.id;
    }

    query(options) {
        if (this.debug) { console.log('query: ' + JSON.stringify(options)); }

        let activeTargets: { [key:string]:any; } = {};
        // const activeTargets: any[] = [];
        for (const target of options.targets) {
            if (target.hide) {
                continue;
            }
            activeTargets[target.refId] = target;
        }
        options = _.clone(options);

        const query = this.buildQueryParameters(options);
        if (query.targets.length <= 0) {
            return this.$q.when({data: []});
        }
        return this.doRequest({data: query})
            .then(result => {

                if (this.debug) { console.log('results pre-table/timeseries: ' + JSON.stringify(result)); }

                const datas: any[] = [];

                _.each(result.data.results, (results, refId) => {
                    if (this.debug) { console.log('single result pre-table/timeseries: ' + JSON.stringify(results)); }

                    if (results.tables && results.tables.length > 0) {
                        const data = results.tables[0];
                        data.type = 'table';
                        datas.push(data);
                    }

                    if (results.series && results.series.length > 0) {

                        _.each(results.series, (series, index) => {
                            const legendFormat = activeTargets[refId].legendFormat;

                            // use legendFormat if set und return renderedLegendFormat instead of series.name
                            if (!_.isEmpty(legendFormat)) {
                                const renderedLegendFormat = this.renderTemplate(this.templateSrv.replace(legendFormat), series.tags);
                                datas.push({target: renderedLegendFormat, datapoints: series.points});
                            } else {
                                datas.push({target: series.name, datapoints: series.points});
                            }
                        });
                    }
                });
                if (this.debug) { console.log('result query: ' + JSON.stringify({data: datas})); }
                return {_request: result._request, data: datas,}
            });
    }

    renderTemplate(aliasPattern, aliasData) {
        const aliasRegex = /\{\{\s*(.+?)\s*\}\}/g;
        return aliasPattern.replace(aliasRegex, function (match, g1) {
            if (aliasData[g1]) {
                return aliasData[g1];
            }
            return g1;
        });
    }

    testDatasource() {
        if (this.debug) { console.log('testDatasource'); }
        return this.backendSrv.datasourceRequest({
            url: '/api/tsdb/query',
            method: 'POST',
            data: {
                queries: [
                    {
                        type: 'test',
                        refId: 'test',
                        datasourceId: this.id,
                    },
                ],
            },
        }).then(response => {
            if (response.status === 200) {
                return {status: 'success', message: 'Data source is working', title: 'Success'};
            }
            return {
                status: 'error',
                message: 'Data source is not working: ' + response.message,
                title: 'Error',
            };
        });
    }

    metricFindQuery(query) {
        if (this.debug) { console.log('metricFindQuery: ' + JSON.stringify(query)); }
        return this.doFindQuery({
            data: {
                targets:
                    [{
                        target: this.templateSrv.replace(query, null, 'regex'),
                        format: 'timeseries',
                        type: 'keys',
                        refId: 'keys',
                        datasourceId: this.id,
                    }],
            },
        }).then(result => {
            const results = result.data.results['keys'];
            return map(results.series, (d) => {
                return {text: d.name, value: d.name};
            });
        });
    }

    doFindQuery(options) {
        if (this.debug) { console.log('doFindQuery: ' + JSON.stringify(options));}
        return this.backendSrv.datasourceRequest({
            url: '/api/tsdb/query',
            method: 'POST',
            data: {
                queries: options.data.targets,
            },
        }).then(result => {
            if (this.debug) { console.log('doFindQuery result: ' + JSON.stringify(result));}
            return result;
        });
    }

    doRequest(options) {
        if (this.debug) { console.log('doRequest: ' + JSON.stringify(options));}

        const data = {
            from: '',
            to: '',
            queries: options.data.targets,
        };
        if (options.data.range) {
            data.from = options.data.range.from.valueOf().toString();
            data.to = options.data.range.to.valueOf().toString();
        }

        return this.backendSrv.datasourceRequest({
            url: '/api/tsdb/query',
            method: 'POST',
            data,
        }).then(result => {
            if (this.debug) { console.log('doRequest result: ' + JSON.stringify(result));}
            return result;
        });
    }

    buildQueryParameters(options) {
        if (this.debug) { console.log('buildQueryParameters: ' + JSON.stringify(options));}

        options.targets = _.filter(options.targets, target => {
            return target.target !== '' && !target.hide;
        });

        options.targets = _.map(options.targets, target => {
            return {
                target: this.templateSrv.replace(target.target, options.scopedVars, 'regex'),
                format: target.format || 'timeseries',
                type: target.type || 'get',
                columns: target.columns || '',
                refId: target.refId,
                hide: target.hide,
                datasourceId: this.id,
            };
        });
        return options;
    }

}

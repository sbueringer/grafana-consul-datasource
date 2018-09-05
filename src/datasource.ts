///<reference path="../node_modules/grafana-sdk-mocks/app/headers/common.d.ts" />
import _ from 'lodash';
import map from 'lodash/map';
import isObject from 'lodash/isObject';
import filter from 'lodash/filter';
import isUndefined from 'lodash/isUndefined';

export class ConsulDatasource {

  name: string;
  id: string;

    /** @ngInject **/
  constructor(instanceSettings, private $q, private backendSrv, private templateSrv) {
    this.name = instanceSettings.name;
    this.id = instanceSettings.id;
  }

  query(options) {
    console.log('query: ' + JSON.stringify(options));

    const activeTargets: any[] = [];
    for (const target of options.targets) {
      if (target.hide) {
        continue;
      }
      activeTargets.push(target);
    }
    options = _.clone(options);

    const query = this.buildQueryParameters(options);
    if (query.targets.length <= 0) {
      return this.$q.when({ data: [] });
    }
    return this.doRequest({ data: query })
            .then(result => {

              const results = result.data.results[''];

              console.log('results pre-table/timeseries: ' + JSON.stringify(results));

              if (results.tables && results.tables.length > 0) {
                result.data = results.tables;
                result.data[0].type = 'table';
                console.log('query table result: ' + JSON.stringify(result));
                return result;
              }

              if (results.series && results.series.length > 0) {
                const data: any[] = [];
                _.each(results.series, (series, index) => {
                  const legendFormat = activeTargets[index].legendFormat;

                    // use legendFormat if set und return renderedLegendFormat instead of series.name
                  if (!_.isEmpty(legendFormat)) {
                    const renderedLegendFormat = this.renderTemplate(this.templateSrv.replace(legendFormat), series.tags);
                    data.push({ target: renderedLegendFormat, datapoints: series.points });
                  } else {
                    data.push({ target: series.name, datapoints: series.points });
                  }
                });
                result.data = data;
                console.log('query timeseries result: ' + JSON.stringify(result));
                return result;
              }
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
    console.log('testDatasource');
    return this.backendSrv.datasourceRequest({
      url: '/api/tsdb/query',
      method: 'POST',
      data: {
        queries: [
          {
            type: 'test',
            datasourceId: this.id,
          },
        ],
      },
    }).then(response => {
      if (response.status === 200) {
        return { status: 'success', message: 'Data source is working', title: 'Success' };
      }
      return {
        status: 'error',
        message: 'Data source is not working: ' + response.message,
        title: 'Error',
      };
    });
  }

  metricFindQuery(query) {
    console.log('metricFindQuery: ' + JSON.stringify(query));
    return this.doFindQuery({
      data: {
        targets:
        [{
          target: this.templateSrv.replace(query, null, 'regex'),
          format: 'timeseries',
          type: 'keys',
          refId: '',
          datasourceId: this.id,
        }],
      },
    }).then(result => {
      const results = result.data.results[''];
      return map(results.series, (d) => {
        return { text: d.name, value: d.name };
      });
    });
  }

  doFindQuery(options) {
    console.log('doFindQuery: ' + JSON.stringify(options));
    return this.backendSrv.datasourceRequest({
      url: '/api/tsdb/query',
      method: 'POST',
      data: {
        queries: options.data.targets,
      },
    }).then(result => {
      console.log('doFindQuery result: ' + JSON.stringify(result));
      return result;
    });
  }

  doRequest(options) {
    console.log('doRequest: ' + JSON.stringify(options));

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
      console.log('doRequest result: ' + JSON.stringify(result));
      return result;
    });
  }

  buildQueryParameters(options) {
    console.log('buildQueryParameters: ' + JSON.stringify(options));

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

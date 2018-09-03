import { ConsulDatasource } from './datasource';
import lodash from 'lodash';
import map from 'lodash/map';

export class ConsulCompleter {
  labelQueryCache: any;
  labelNameCache: any;
  labelValueCache: any;
  templateVariableCompletions: any;

  constructor(private datasource: ConsulDatasource, private templateSrv) {
    this.labelQueryCache = {};
    this.labelNameCache = {};
    this.labelValueCache = {};
    this.templateVariableCompletions = this.templateSrv.variables.map((variable) => {
      return {
        caption: '${' + variable.name + '}',
        value: '${' + variable.name + '}',
        meta: 'variable',
        score: Number.MAX_VALUE - 1,
      };
    });
  }

  getCompletions(editor, session, pos, prefix, callback) {
    const wrappedCallback = (err, completions) => {
      completions = completions.concat(this.templateVariableCompletions);
      return callback(err, completions);
    };

    const token = session.getTokenAt(pos.row, pos.column);
    const renderedToken = this.templateSrv.replace(token.value, null, 'regex');

    this.datasource.doFindQuery({
      data: {
        targets:
        [{
          target: renderedToken,
          format: 'timeseries',
          type: 'keys',
          refId: '',
          datasourceId: this.datasource.id,
        }],
      },
    }).then(result => {
      const results = result.data.results[''];
      wrappedCallback(null, map(results.series, (d) => {
        const completion = d.name.slice(renderedToken.length);
        return { caption: d.name, value: completion, meta: 'key', score: Number.MAX_VALUE };
      }));
      return result;
    });
  }
}

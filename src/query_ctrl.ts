import { QueryCtrl } from 'grafana/app/plugins/sdk';
import { ConsulCompleter } from './completer';

export class ConsulDatasourceQueryCtrl extends QueryCtrl {
  static templateUrl = 'partials/query.editor.html';

  private formats: any;
  private types: any;

    /** @ngInject **/
  constructor($scope, $injector, private templateSrv) {
    super($scope, $injector);

        // special handling when in table panel
    if (!this.target.format) {
      this.target.format = this.panelCtrl.panel.type === 'table' ? 'table' : 'timeseries';
    }

    this.target.target = this.target.target || '';
    this.target.format = this.target.format || 'timeseries';
    this.target.type = this.target.type || 'get';
    this.target.columns = this.target.columns || '';
    this.target.data = this.target.data || '';

    this.formats = [
            { text: 'Time series', value: 'timeseries' },
            { text: 'Table', value: 'table' },
    ];
    this.types = [
            { text: 'get value', value: 'get' },
            { text: 'get direct subkeys', value: 'keys' },
            { text: 'get subkeys as tags', value: 'tags' },
            { text: 'get subkeys recursive as tags', value: 'tagsrec' },
    ];
  }

  getCompleter(query) {
    return new ConsulCompleter(this.datasource, this.templateSrv);
  }

  refreshMetricData() {
    this.panelCtrl.refresh();
  }
}

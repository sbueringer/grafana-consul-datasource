import { ConsulDatasource } from './datasource';
import { ConsulDatasourceQueryCtrl } from './query_ctrl';

class ConsulConfigCtrl {
  static templateUrl = 'partials/config.html';
}

class ConsulQueryOptionsCtrl {
  static templateUrl = 'partials/query.options.html';
}

export {
    ConsulDatasource as Datasource,
    ConsulConfigCtrl as ConfigCtrl,
    ConsulQueryOptionsCtrl as QueryOptionsCtrl,
    ConsulDatasourceQueryCtrl as QueryCtrl,
};

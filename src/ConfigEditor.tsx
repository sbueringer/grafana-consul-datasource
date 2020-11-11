import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MyDataSourceOptions, MySecureJsonData } from './types';

const { SecretFormField, FormField } = LegacyForms;

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions> {}

interface State {}

export class ConfigEditor extends PureComponent<Props, State> {
  onConsulAddrChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      consulAddr: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  // Secure field (only sent to the backend)
  onConsulTakenChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonData: {
        consulToken: event.target.value,
      },
    });
  };

  onResetConsulToken = () => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        consulToken: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        consulToken: '',
      },
    });
  };

  render() {
    const { options } = this.props;
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as MySecureJsonData;

    return (
      <div className="gf-form-group">
        <div className="gf-form">
          <FormField
            label="Address"
            labelWidth={6}
            inputWidth={20}
            onChange={this.onConsulAddrChange}
            value={jsonData.consulAddr || ''}
            placeholder="http://localhost:8500"
            tooltip="Specify a complete HTTP URL. This is usually one of the addresses specified in the Consul configuration under `addresses`. The default value when running Consul locally is `http://localhost:8500`. More details can be found in the Consul documentation. Consul is accessed by the Consul plugin backend, this means the URL needs to be accessible from the Grafana server."
          />
        </div>

        <div className="gf-form-inline">
          <div className="gf-form">
            <SecretFormField
              isConfigured={(secureJsonFields && secureJsonFields.consulToken) as boolean}
              value={secureJsonData.consulToken || ''}
              label="Token"
              placeholder="CONSUL_TOKEN"
              labelWidth={6}
              inputWidth={20}
              onReset={this.onResetConsulToken}
              onChange={this.onConsulTakenChange}
              tooltip=" If Consul Token is set, it has to be a valid Consul Token which is able to read the data you want to access.
            If Consul Token is not set, no token will be set on the Consul client."
            />
          </div>
        </div>
      </div>
    );
  }
}

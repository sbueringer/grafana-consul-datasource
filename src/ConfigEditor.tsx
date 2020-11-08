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
            />
          </div>
        </div>
      </div>
    );
  }
}

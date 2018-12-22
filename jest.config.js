module.exports = {
    verbose: true,
    globals: {
        'ts-jest': {
            tsConfig: 'tsconfig.jest.json',
            babelConfig: true,
        },
    },
    moduleNameMapper: {
        'app/plugins/sdk': '<rootDir>/node_modules/grafana-sdk-mocks/app/plugins/sdk.ts',
    },
    transformIgnorePatterns: [
        'node_modules/(?!(grafana-sdk-mocks))',
    ],
    transform: {
        "^.+\\.tsx?$": "ts-jest"
    },
    testRegex: '(\\.|/)([jt]est)\\.ts$',
    moduleFileExtensions: [
        'js',
        'json',
        'jsx',
        'ts',
        'tsx',
    ],
    collectCoverageFrom: [
        'src/*.ts',
        '!**/node_modules/**',
        '!**/vendor/**',
    ],
    coverageDirectory: '<rootDir>/coverage',
    coverageReporters: [
        'json',
        'lcov',
        'text',
    ],
    preset: 'ts-jest',
    testMatch: null,
}

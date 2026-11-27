const { withAppBuildGradle } = require('expo/config-plugins');

module.exports = function withAndroidObjectPaths(config) {
  if (process.platform !== 'win32') return config;

  return withAppBuildGradle(config, (config) => {
    const gradle = config.modResults;
    const argument = '-DCMAKE_OBJECT_PATH_MAX=240';
    if (gradle.contents.includes(argument)) return config;

    // The SDK's Ninja can fail to create nested object directories at MAX_PATH.
    const defaultConfig = /^([ \t]*)defaultConfig \{[ \t]*$/m;
    gradle.contents = gradle.contents.replace(defaultConfig, (line, indent) => [
      line,
      indent + '    externalNativeBuild {',
      indent + '        cmake {',
      indent + '            arguments "' + argument + '"',
      indent + '        }',
      indent + '    }',
    ].join('\n'));
    return config;
  });
};

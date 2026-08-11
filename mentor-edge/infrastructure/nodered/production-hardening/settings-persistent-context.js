"use strict";

const baseSettingsPath = "/data/settings.mentor-base.js";
const settings = require(baseSettingsPath);

if (
    Object.prototype.hasOwnProperty.call(settings, "contextStorage") &&
    settings.contextStorage !== undefined
) {
    throw new Error(
        `${baseSettingsPath} ya define contextStorage; se requiere revisión manual`
    );
}

settings.contextStorage = {
    default: {
        module: "localfilesystem",
        config: {
            dir: "/data",
            base: "mentor-context",
            cache: true,
            flushInterval: 5
        }
    }
};

module.exports = settings;

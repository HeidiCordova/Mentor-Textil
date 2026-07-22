import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.mentor.tablet',
  appName: 'Mentor Tablet',
  webDir: 'dist',
  server: {
    androidScheme: 'https'
  },
  plugins: {
    Network: {},
    Preferences: {}
  }
}

export default config

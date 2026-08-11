import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.mentor.tablet',
  appName: 'Mentor Textil',
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

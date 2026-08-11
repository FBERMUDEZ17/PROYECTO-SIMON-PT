export type AuthStackParamList = {
  Login: undefined;
  Register: undefined;
  ForgotPassword: undefined;
};

export type MainTabParamList = {
  Dashboard: undefined;
  Alerts: undefined;
  Settings: undefined;
};

export type MainStackParamList = {
  Tabs: undefined;
  VehicleDetail: { vehicleId: string };
};

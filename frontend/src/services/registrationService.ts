export const validateProductKey = (key: string): boolean => {
  const keyRegex = /^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$/;
  return keyRegex.test(key);
};

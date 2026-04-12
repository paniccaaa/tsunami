import { useState, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import type { AttackConfig } from '../types/api';
import { uploadProtoFile } from '../services/api';

interface AttackFormProps {
  onStart: (config: AttackConfig) => void;
  onStop: () => void;
  isRunning: boolean;
  isLoading: boolean;
}

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'];

// Validation patterns
const RATE_REGEX = /^\d+\/\d+(ms|s|m|h)$/;
const DURATION_REGEX = /^(\d+)(ms|s|m|h)$/;
const HOST_PORT_REGEX = /^[^:]+:\d+$/;

// Tooltip component
function Tooltip({ text }: { text: string }) {
  const [show, setShow] = useState(false);

  return (
    <div className="relative inline-block ml-1">
      <button
        type="button"
        className="w-4 h-4 rounded-full bg-gray-600 text-gray-300 text-xs hover:bg-gray-500 focus:outline-none"
        onMouseEnter={() => setShow(true)}
        onMouseLeave={() => setShow(false)}
        onClick={() => setShow(!show)}
      >
        ?
      </button>
      {show && (
        <div className="absolute z-10 w-64 p-2 mt-1 text-sm text-gray-200 bg-gray-700 rounded-md shadow-lg -left-28 top-6">
          {text}
        </div>
      )}
    </div>
  );
}

// Label with tooltip
function Label({ text, tooltip }: { text: string; tooltip: string }) {
  return (
    <label className="flex items-center text-sm font-medium text-gray-300 mb-1">
      {text}
      <Tooltip text={tooltip} />
    </label>
  );
}

// Validation error message
function ValidationError({ message }: { message: string | null }) {
  if (!message) return null;
  return <p className="text-red-400 text-xs mt-1">{message}</p>;
}

export function AttackForm({ onStart, onStop, isRunning, isLoading }: AttackFormProps) {
  const { t } = useTranslation();

  // Protocol selector
  const [protocol, setProtocol] = useState<'http' | 'grpc'>('http');

  // Shared fields
  const [rate, setRate] = useState('100/1s');
  const [duration, setDuration] = useState('30s');
  const [timeout, setTimeout] = useState('10s');
  const [workers, setWorkers] = useState(50);
  const [connections, setConnections] = useState(100);

  // HTTP fields
  const [url, setUrl] = useState('');
  const [method, setMethod] = useState('GET');
  const [body, setBody] = useState('');
  const [headers, setHeaders] = useState<string[]>([]);
  const [newHeader, setNewHeader] = useState('');

  // gRPC fields
  const [grpcTarget, setGrpcTarget] = useState('');
  const [grpcService, setGrpcService] = useState('');
  const [grpcMethod, setGrpcMethod] = useState('');
  const [grpcData, setGrpcData] = useState('{}');
  const [grpcProto, setGrpcProto] = useState('');
  const [protoFileName, setProtoFileName] = useState('');
  const [protoUploadError, setProtoUploadError] = useState<string | null>(null);
  const [isUploadingProto, setIsUploadingProto] = useState(false);
  const protoInputRef = useRef<HTMLInputElement>(null);
  const [grpcMetadata, setGrpcMetadata] = useState<string[]>([]);
  const [newMeta, setNewMeta] = useState('');
  const [insecure, setInsecure] = useState(false);

  // Validation
  const errors = useMemo(() => {
    const errs: Record<string, string | null> = {
      url: null,
      grpcTarget: null,
      grpcService: null,
      grpcMethod: null,
      grpcData: null,
      rate: null,
      duration: null,
      timeout: null,
      workers: null,
      connections: null,
    };

    if (protocol === 'http') {
      if (url && !url.startsWith('http://') && !url.startsWith('https://')) {
        errs.url = t('form.url.error');
      }
    } else {
      if (grpcTarget && !HOST_PORT_REGEX.test(grpcTarget)) {
        errs.grpcTarget = t('form.grpcTarget.error');
      }
      if (grpcService === '') {
        errs.grpcService = null; // empty is fine until submit
      }
      if (grpcMethod === '') {
        errs.grpcMethod = null;
      }
      if (grpcData) {
        try {
          JSON.parse(grpcData);
        } catch {
          errs.grpcData = t('form.grpcData.error');
        }
      }
    }

    if (rate && !RATE_REGEX.test(rate)) {
      errs.rate = t('form.rate.error');
    }

    if (duration && duration !== '0' && !DURATION_REGEX.test(duration)) {
      errs.duration = t('form.duration.error');
    }

    if (timeout && !DURATION_REGEX.test(timeout)) {
      errs.timeout = t('form.timeout.error');
    }

    if (workers !== null && workers < 1) {
      errs.workers = t('form.workers.error');
    }

    if (connections !== null && connections < 1) {
      errs.connections = t('form.connections.error');
    }

    return errs;
  }, [protocol, url, grpcTarget, grpcService, grpcMethod, grpcData, rate, duration, timeout, workers, connections, t]);

  const hasErrors = Object.values(errors).some((e) => e !== null);
  const isFormValid = protocol === 'http'
    ? url && !hasErrors
    : grpcTarget && grpcService && grpcMethod && !hasErrors;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!isFormValid) return;

    const shared = {
      protocol,
      rate,
      duration,
      timeout,
      workers: workers || 50,
      connections: connections || 100,
    };

    if (protocol === 'grpc') {
      onStart({
        ...shared,
        grpc_target: grpcTarget,
        grpc_service: grpcService,
        grpc_method: grpcMethod,
        grpc_data: grpcData || '{}',
        grpc_proto: grpcProto || undefined,
        grpc_metadata: grpcMetadata.length > 0 ? grpcMetadata : undefined,
        insecure,
      });
    } else {
      onStart({
        ...shared,
        url,
        method,
        body: body || undefined,
        headers: headers.length > 0 ? headers : undefined,
      });
    }
  };

  const addHeader = () => {
    if (newHeader.includes(':')) {
      setHeaders([...headers, newHeader]);
      setNewHeader('');
    }
  };

  const removeHeader = (index: number) => {
    setHeaders(headers.filter((_, i) => i !== index));
  };

  const addMeta = () => {
    if (newMeta.includes(':')) {
      setGrpcMetadata([...grpcMetadata, newMeta]);
      setNewMeta('');
    }
  };

  const removeMeta = (index: number) => {
    setGrpcMetadata(grpcMetadata.filter((_, i) => i !== index));
  };

  const handleProtoFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setProtoUploadError(null);
    setIsUploadingProto(true);
    try {
      const result = await uploadProtoFile(file);
      setGrpcProto(result.path);
      setProtoFileName(result.name);
    } catch (err) {
      setProtoUploadError(err instanceof Error ? err.message : t('form.grpcProto.uploadError'));
      setGrpcProto('');
      setProtoFileName('');
    } finally {
      setIsUploadingProto(false);
      if (protoInputRef.current) protoInputRef.current.value = '';
    }
  };

  return (
    <form onSubmit={handleSubmit} className="bg-gray-800 rounded-lg p-6 space-y-4">
      <h2 className="text-xl font-bold text-white mb-4">{t('form.title')}</h2>

      {/* Protocol Toggle */}
      <div className="flex items-center gap-2 pb-2 border-b border-gray-700">
        <span className="text-sm font-medium text-gray-300">{t('form.protocol.label')}:</span>
        <div className="flex rounded-md overflow-hidden border border-gray-600">
          <button
            type="button"
            onClick={() => setProtocol('http')}
            disabled={isRunning}
            className={`px-4 py-1.5 text-sm font-medium transition-colors disabled:opacity-50 ${
              protocol === 'http'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
            }`}
          >
            {t('form.protocol.http')}
          </button>
          <button
            type="button"
            onClick={() => setProtocol('grpc')}
            disabled={isRunning}
            className={`px-4 py-1.5 text-sm font-medium transition-colors disabled:opacity-50 ${
              protocol === 'grpc'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
            }`}
          >
            {t('form.protocol.grpc')}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {protocol === 'http' ? (
          <>
            {/* HTTP fields */}
            <div className="md:col-span-2">
              <Label text={t('form.url.label')} tooltip={t('form.url.tooltip')} />
              <input
                type="url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder={t('form.url.placeholder')}
                required
                disabled={isRunning}
                className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
                  errors.url ? 'border-red-500' : 'border-gray-600'
                }`}
              />
              <ValidationError message={errors.url} />
            </div>

            <div>
              <Label text={t('form.method.label')} tooltip={t('form.method.tooltip')} />
              <select
                value={method}
                onChange={(e) => setMethod(e.target.value)}
                disabled={isRunning}
                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
              >
                {HTTP_METHODS.map((m) => (
                  <option key={m} value={m}>{m}</option>
                ))}
              </select>
            </div>
          </>
        ) : (
          <>
            {/* gRPC fields */}
            <div className="md:col-span-2">
              <Label text={t('form.grpcTarget.label')} tooltip={t('form.grpcTarget.tooltip')} />
              <input
                type="text"
                value={grpcTarget}
                onChange={(e) => setGrpcTarget(e.target.value)}
                placeholder={t('form.grpcTarget.placeholder')}
                required
                disabled={isRunning}
                className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
                  errors.grpcTarget ? 'border-red-500' : 'border-gray-600'
                }`}
              />
              <ValidationError message={errors.grpcTarget} />
            </div>

            <div>
              <Label text={t('form.grpcService.label')} tooltip={t('form.grpcService.tooltip')} />
              <input
                type="text"
                value={grpcService}
                onChange={(e) => setGrpcService(e.target.value)}
                placeholder={t('form.grpcService.placeholder')}
                required
                disabled={isRunning}
                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
              />
            </div>

            <div>
              <Label text={t('form.grpcMethod.label')} tooltip={t('form.grpcMethod.tooltip')} />
              <input
                type="text"
                value={grpcMethod}
                onChange={(e) => setGrpcMethod(e.target.value)}
                placeholder={t('form.grpcMethod.placeholder')}
                required
                disabled={isRunning}
                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
              />
            </div>

            <div className="md:col-span-2">
              <Label text={t('form.grpcData.label')} tooltip={t('form.grpcData.tooltip')} />
              <textarea
                value={grpcData}
                onChange={(e) => setGrpcData(e.target.value)}
                placeholder={t('form.grpcData.placeholder')}
                rows={3}
                disabled={isRunning}
                className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 font-mono text-sm ${
                  errors.grpcData ? 'border-red-500' : 'border-gray-600'
                }`}
              />
              <ValidationError message={errors.grpcData} />
            </div>

            <div className="md:col-span-2">
              <Label text={t('form.grpcProto.label')} tooltip={t('form.grpcProto.tooltip')} />
              <input
                ref={protoInputRef}
                type="file"
                accept=".proto"
                className="hidden"
                disabled={isRunning || isUploadingProto}
                onChange={handleProtoFileChange}
              />
              <div className="flex gap-2 items-center">
                <button
                  type="button"
                  disabled={isRunning || isUploadingProto}
                  onClick={() => protoInputRef.current?.click()}
                  className="px-3 py-2 bg-gray-600 text-white text-sm rounded-md hover:bg-gray-500 disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
                >
                  {isUploadingProto ? t('form.grpcProto.uploading') : t('form.grpcProto.browse')}
                </button>
                {protoFileName ? (
                  <div className="flex-1 flex items-center gap-2 px-3 py-2 bg-gray-700 border border-gray-600 rounded-md">
                    <span className="flex-1 text-sm text-gray-200 truncate">{protoFileName}</span>
                    <button
                      type="button"
                      disabled={isRunning}
                      onClick={() => { setGrpcProto(''); setProtoFileName(''); setProtoUploadError(null); }}
                      className="text-gray-400 hover:text-gray-200 disabled:opacity-50 text-xs"
                      aria-label={t('form.grpcProto.clear')}
                    >
                      ✕
                    </button>
                  </div>
                ) : (
                  <input
                    type="text"
                    value={grpcProto}
                    onChange={(e) => { setGrpcProto(e.target.value); setProtoUploadError(null); }}
                    placeholder={t('form.grpcProto.placeholder')}
                    disabled={isRunning || isUploadingProto}
                    className="flex-1 px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                  />
                )}
              </div>
              {protoUploadError && (
                <p className="text-red-400 text-xs mt-1">{protoUploadError}</p>
              )}
            </div>

            <div className="md:col-span-2">
              <Label text={t('form.grpcMetadata.label')} tooltip={t('form.grpcMetadata.tooltip')} />
              <div className="flex gap-2 mb-2">
                <input
                  type="text"
                  value={newMeta}
                  onChange={(e) => setNewMeta(e.target.value)}
                  placeholder={t('form.grpcMetadata.placeholder')}
                  disabled={isRunning}
                  className="flex-1 px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                  onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addMeta())}
                />
                <button
                  type="button"
                  onClick={addMeta}
                  disabled={isRunning || !newMeta.includes(':')}
                  className="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-500 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {t('form.grpcMetadata.add')}
                </button>
              </div>
              {grpcMetadata.length > 0 && (
                <div className="space-y-1">
                  {grpcMetadata.map((meta, index) => (
                    <div key={index} className="flex items-center gap-2 text-sm">
                      <code className="flex-1 px-2 py-1 bg-gray-700 rounded text-gray-300">{meta}</code>
                      <button
                        type="button"
                        onClick={() => removeMeta(index)}
                        disabled={isRunning}
                        className="text-red-400 hover:text-red-300 disabled:opacity-50"
                      >
                        {t('form.grpcMetadata.remove')}
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="md:col-span-2">
              <label className="flex items-center gap-2 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={insecure}
                  onChange={(e) => setInsecure(e.target.checked)}
                  disabled={isRunning}
                  className="w-4 h-4 rounded border-gray-600 bg-gray-700 text-blue-600 focus:ring-blue-500 disabled:opacity-50"
                />
                <span className="text-sm font-medium text-gray-300">{t('form.insecure.label')}</span>
                <Tooltip text={t('form.insecure.tooltip')} />
              </label>
            </div>
          </>
        )}

        {/* Shared fields: Rate, Duration, Timeout, Workers, Connections */}
        <div>
          <Label text={t('form.rate.label')} tooltip={t('form.rate.tooltip')} />
          <input
            type="text"
            value={rate}
            onChange={(e) => setRate(e.target.value)}
            placeholder={t('form.rate.placeholder')}
            disabled={isRunning}
            className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
              errors.rate ? 'border-red-500' : 'border-gray-600'
            }`}
          />
          <ValidationError message={errors.rate} />
        </div>

        <div>
          <Label text={t('form.duration.label')} tooltip={t('form.duration.tooltip')} />
          <input
            type="text"
            value={duration}
            onChange={(e) => setDuration(e.target.value)}
            placeholder={t('form.duration.placeholder')}
            disabled={isRunning}
            className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
              errors.duration ? 'border-red-500' : 'border-gray-600'
            }`}
          />
          <ValidationError message={errors.duration} />
        </div>

        <div>
          <Label text={t('form.timeout.label')} tooltip={t('form.timeout.tooltip')} />
          <input
            type="text"
            value={timeout}
            onChange={(e) => setTimeout(e.target.value)}
            placeholder={t('form.timeout.placeholder')}
            disabled={isRunning}
            className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
              errors.timeout ? 'border-red-500' : 'border-gray-600'
            }`}
          />
          <ValidationError message={errors.timeout} />
        </div>

        <div>
          <Label text={t('form.workers.label')} tooltip={t('form.workers.tooltip')} />
          <input
            type="number"
            value={workers || ''}
            onChange={(e) => setWorkers(e.target.value === '' ? 0 : parseInt(e.target.value, 10) || 0)}
            min="1"
            placeholder={t('form.workers.placeholder')}
            disabled={isRunning}
            className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
              errors.workers ? 'border-red-500' : 'border-gray-600'
            }`}
          />
          <ValidationError message={errors.workers} />
        </div>

        <div>
          <Label text={t('form.connections.label')} tooltip={t('form.connections.tooltip')} />
          <input
            type="number"
            value={connections || ''}
            onChange={(e) => setConnections(e.target.value === '' ? 0 : parseInt(e.target.value, 10) || 0)}
            min="1"
            placeholder={t('form.connections.placeholder')}
            disabled={isRunning}
            className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
              errors.connections ? 'border-red-500' : 'border-gray-600'
            }`}
          />
          <ValidationError message={errors.connections} />
        </div>

        {/* HTTP body (shown only for POST/PUT/PATCH in HTTP mode) */}
        {protocol === 'http' && (method === 'POST' || method === 'PUT' || method === 'PATCH') && (
          <div className="md:col-span-2">
            <Label text={t('form.body.label')} tooltip={t('form.body.tooltip')} />
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder={t('form.body.placeholder')}
              rows={3}
              disabled={isRunning}
              className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 font-mono text-sm"
            />
          </div>
        )}

        {/* HTTP headers */}
        {protocol === 'http' && (
          <div className="md:col-span-2">
            <Label text={t('form.headers.label')} tooltip={t('form.headers.tooltip')} />
            <div className="flex gap-2 mb-2">
              <input
                type="text"
                value={newHeader}
                onChange={(e) => setNewHeader(e.target.value)}
                placeholder={t('form.headers.placeholder')}
                disabled={isRunning}
                className="flex-1 px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addHeader())}
              />
              <button
                type="button"
                onClick={addHeader}
                disabled={isRunning || !newHeader.includes(':')}
                className="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {t('form.headers.add')}
              </button>
            </div>
            {headers.length > 0 && (
              <div className="space-y-1">
                {headers.map((header, index) => (
                  <div key={index} className="flex items-center gap-2 text-sm">
                    <code className="flex-1 px-2 py-1 bg-gray-700 rounded text-gray-300">{header}</code>
                    <button
                      type="button"
                      onClick={() => removeHeader(index)}
                      disabled={isRunning}
                      className="text-red-400 hover:text-red-300 disabled:opacity-50"
                    >
                      {t('form.headers.remove')}
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      <div className="flex gap-4 pt-4">
        {!isRunning ? (
          <button
            type="submit"
            disabled={isLoading || !isFormValid}
            className="flex-1 px-6 py-3 bg-blue-600 text-white font-semibold rounded-md hover:bg-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-gray-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isLoading ? t('form.starting') : t('form.startButton')}
          </button>
        ) : (
          <button
            type="button"
            onClick={onStop}
            disabled={isLoading}
            className="flex-1 px-6 py-3 bg-red-600 text-white font-semibold rounded-md hover:bg-red-500 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 focus:ring-offset-gray-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isLoading ? t('form.stopping') : t('form.stopButton')}
          </button>
        )}
      </div>
    </form>
  );
}

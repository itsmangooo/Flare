import 'dart:io';

import 'package:image_picker/image_picker.dart';
import 'package:path_provider/path_provider.dart';

abstract final class CustomBackgroundStorage {
  static const _directoryName = 'appearance';
  static const _fileStem = 'custom-background';

  static Future<String> persist(XFile image) async {
    final directory = await _backgroundDirectory();
    await directory.create(recursive: true);

    final extension = _safeExtension(image.name);
    final separator = Platform.pathSeparator;
    final destination = File('${directory.path}$separator$_fileStem$extension');
    final temporary = File('${directory.path}$separator$_fileStem.tmp');

    if (await temporary.exists()) await temporary.delete();
    await image.saveTo(temporary.path);
    if (await destination.exists()) await destination.delete();
    await temporary.rename(destination.path);

    await for (final entry in directory.list()) {
      if (entry is File && entry.path != destination.path) {
        await entry.delete();
      }
    }
    return destination.path;
  }

  static Future<void> clear() async {
    final directory = await _backgroundDirectory();
    if (!await directory.exists()) return;
    await for (final entry in directory.list()) {
      if (entry is File) await entry.delete();
    }
  }

  static Future<Directory> _backgroundDirectory() async {
    final support = await getApplicationSupportDirectory();
    return Directory('${support.path}${Platform.pathSeparator}$_directoryName');
  }

  static String _safeExtension(String name) {
    final dot = name.lastIndexOf('.');
    if (dot < 0) return '.img';
    return switch (name.substring(dot).toLowerCase()) {
      '.jpg' ||
      '.jpeg' ||
      '.png' ||
      '.webp' ||
      '.gif' ||
      '.heic' ||
      '.heif' => name.substring(dot).toLowerCase(),
      _ => '.img',
    };
  }
}

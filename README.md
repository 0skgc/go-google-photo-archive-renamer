# はじめに

GooglePhotoの写真や動画を、アーカイブデータとしてダウンロードしたファイルの名前を撮影日にリネームするためのツールです。

# GooglePhotoアーカイブデータ

以下のように写真・動画ファイルとペアで、メタデータJsonファイルがあります。

- hoge.jpg
- hoge.jpg.json

写真・動画ファイル名が重複した場合は少し変則的な名前のJsonファイルになります。

- hoge(1).jpg
- hoge.jpg(1).json

## メタデータJSONファイルの構造
`photoTakenTime` に撮影日情報が残っているためこれを利用します。
写真の場合は`EXIF`というメタ情報から取得してもよいのですが、動画の場合はメタ情報から取得するのは困難です。
写真・動画ともに同じ仕組みで撮影日を取得するのは`photoTakenTime`を利用するのがベターです。

    {
      "title": "hoge.jpg",
      "description": "",
      "imageViews": "0",
      "creationTime": {
        "timestamp": "1576984416",
        "formatted": "2019/12/22 3:13:36 UTC"
      },
      "modificationTime": {
        "timestamp": "1576985027",
        "formatted": "2019/12/22 3:23:47 UTC"
      },
      "geoData": {
        "latitude": 0.0,
        "longitude": 0.0,
        "altitude": 0.0,
        "latitudeSpan": 0.0,
        "longitudeSpan": 0.0
      },
      "geoDataExif": {
        "latitude": 0.0,
        "longitude": 0.0,
        "altitude": 0.0,
        "latitudeSpan": 0.0,
        "longitudeSpan": 0.0
      },
      "photoTakenTime": {
        "timestamp": "1542514086",
        "formatted": "2018/11/18 4:08:06 UTC"
      }
    }

## JSONファイルが無い場合
JSONファイルのペアが無い場合も一部でありました。
JSONファイルが無い場合は、JPEGおよびHEICのEXIF情報を可能な限り読み込みます。

# ツールの使い方
## インストール
`go get github.com/0skgc/go-google-photo-archive-renamer`

## 実行コマンド
`go-google-photo-archive-renamer`

コマンドが見つからない場合は、Go言語のbinディレクトリにパスが通っていないと思われます。

以下を実行して `~/.bash_profile` にパスを追記してください。

    $ echo 'export GOPATH="$HOME/go"' >> ~/.bash_profile
    $ echo 'export PATH="$GOPATH/bin:$PATH"' >> ~/.bash_profile
    $ source ~/.bash_profile

### オプション

- `-target` : [必須]リネームファイルを含むディレクトリを指定します。サブディレクトリは無視します。
- `-dryrun` : 指定するとログ出力のみ行い、実際のリネームは行いません。
- `-h` : ヘルプ

## 実行例
    $ go-google-photo-archive-renamer -target ~/Downloads/Takeout/Google\ Photo/Backup\ Album -dryrun

    2019/12/30 17:20:50 # Rename(dryrun)
    2019/12/30 17:20:50   - From: /Users/user/Downloads/Takeout/Google Photo/Backup Album/hoge.jpg
    2019/12/30 17:20:50   -   To: /Users/user/Downloads/Takeout/Google Photo/Backup Album/20190806-185531-0.jpg
    2019/12/30 17:20:50 # Rename(dryrun)
    2019/12/30 17:20:50   - From: /Users/user/Downloads/Takeout/Google Photo/Backup Album/hoge.jpg.json
    2019/12/30 17:20:50   -   To: /Users/user/Downloads/Takeout/Google Photo/Backup Album/20190806-185531-0.jpg.json

    2019/12/30 17:20:50 # Renamed Ext
    2019/12/30 17:20:50   - .jpg: 2714
    2019/12/30 17:20:50   - .AVI: 1
    2019/12/30 17:20:50   - .gif: 75
    2019/12/30 17:20:50   - .HEIC: 257
    2019/12/30 17:20:50   - .MOV: 26
    2019/12/30 17:20:50   - .jpeg: 4
    2019/12/30 17:20:50   - .mp4: 159
    2019/12/30 17:20:50   - .JPG: 3019
    2019/12/30 17:20:50   - .MP4: 136
    2019/12/30 17:20:50   - .GIF: 1

    2019/12/30 17:20:50 # UnRenamed Ext
    2019/12/30 17:20:50   - .mp4: 4
    2019/12/30 17:20:50   - .JPG: 22
    2019/12/30 17:20:50   - .HEIC: 580
    2019/12/30 17:20:50   - .MOV: 3
    2019/12/30 17:20:50   - .gif: 2
    2019/12/30 17:20:50   - .DS_Store: 1
    2019/12/30 17:20:50   - .jpg: 1010
